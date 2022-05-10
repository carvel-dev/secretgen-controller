// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package generator

import (
	"bytes"
	"context"
	"fmt"

	"github.com/go-logr/logr"
	sgv1alpha1 "github.com/vmware-tanzu/carvel-secretgen-controller/pkg/apis/secretgen/v1alpha1"
	sg2v1alpha1 "github.com/vmware-tanzu/carvel-secretgen-controller/pkg/apis/secretgen2/v1alpha1"
	"github.com/vmware-tanzu/carvel-secretgen-controller/pkg/client2/clientset/versioned/scheme"
	"github.com/vmware-tanzu/carvel-secretgen-controller/pkg/reconciler"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"k8s.io/client-go/util/jsonpath"
)

type SecretTemplateReconciler struct {
	client client.Client
	log    logr.Logger
}

var _ reconcile.Reconciler = &SecretTemplateReconciler{}

func NewSecretTemplateReconciler(client client.Client, log logr.Logger) *SecretTemplateReconciler {
	return &SecretTemplateReconciler{client, log}
}

// AttachWatches adds starts watches this reconciler requires.
func (r *SecretTemplateReconciler) AttachWatches(controller controller.Controller) error {
	//Watch for changes to created Secrets
	if err := controller.Watch(&source.Kind{Type: &corev1.Secret{}}, &handler.EnqueueRequestForOwner{}); err != nil {
		return err
	}
	return controller.Watch(&source.Kind{Type: &sg2v1alpha1.SecretTemplate{}}, &handler.EnqueueRequestForObject{})
}

// Reconcile is the entrypoint for incoming requests from k8s
func (r *SecretTemplateReconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.WithValues("request", request)

	secretTemplate := sg2v1alpha1.SecretTemplate{}
	if err := r.client.Get(ctx, types.NamespacedName{Namespace: request.Namespace, Name: request.Name}, &secretTemplate); err != nil {
		if errors.IsNotFound(err) {
			log.Info("Not found")
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	if secretTemplate.DeletionTimestamp != nil {
		return reconcile.Result{}, nil
	}

	status := &reconciler.Status{
		secretTemplate.Status.GenericStatus,
		func(st sgv1alpha1.GenericStatus) { secretTemplate.Status.GenericStatus = st },
	}

	status.SetReconciling(secretTemplate.ObjectMeta)
	defer r.updateStatus(ctx, &secretTemplate)

	return status.WithReconcileCompleted(r.reconcile(ctx, &secretTemplate))
}

func (r *SecretTemplateReconciler) reconcile(ctx context.Context, secretTemplate *sg2v1alpha1.SecretTemplate) (reconcile.Result, error) {
	//Resolve input resources
	inputResources, err := r.resolveInputResources(ctx, secretTemplate)
	if err != nil {
		return reconcile.Result{}, err
	}

	//Template Secret Data
	secretData := map[string][]byte{}
	for key, expression := range secretTemplate.Spec.JsonPathTemplate.Data {
		valueBuffer, err := jsonPath(expression, inputResources)
		if err != nil {
			//todo jsonpath error
			return reconcile.Result{}, err
		}
		secretData[key] = valueBuffer.Bytes()
	}
	//Template Secret StringData
	secretStringData := map[string]string{}
	for key, expression := range secretTemplate.Spec.JsonPathTemplate.StringData {
		valueBuffer, err := jsonPath(expression, inputResources)
		if err != nil {
			//todo jsonpath error
			return reconcile.Result{}, err
		}
		secretStringData[key] = valueBuffer.String()
	}

	//Create Secret
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretTemplate.GetName(),
			Namespace: secretTemplate.GetNamespace(),
		},
	}

	controllerutil.SetControllerReference(secretTemplate, secret, scheme.Scheme)

	//TODO handle existing secret
	if _, err := controllerutil.CreateOrUpdate(ctx, r.client, secret, func() error {
		secret.ObjectMeta.Labels = secretTemplate.GetLabels()           //TODO do we want these implicitly?
		secret.ObjectMeta.Annotations = secretTemplate.GetAnnotations() //TODO do we want these implicitly?
		secret.StringData = secretStringData
		secret.Data = secretData
		return nil
	}); err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

func (r *SecretTemplateReconciler) updateStatus(ctx context.Context, secretTemplate *sg2v1alpha1.SecretTemplate) error {
	existingSecretTemplate := sg2v1alpha1.SecretTemplate{}
	if err := r.client.Get(ctx, types.NamespacedName{Namespace: secretTemplate.Namespace, Name: secretTemplate.Name}, &existingSecretTemplate); err != nil {
		return fmt.Errorf("Fetching secretTemplate: %s", err)
	}

	existingSecretTemplate.Status = secretTemplate.Status

	if err := r.client.Status().Update(ctx, &existingSecretTemplate); err != nil {
		return fmt.Errorf("Updating secretTemplate status: %s", err)
	}

	return nil
}

func (r *SecretTemplateReconciler) resolveInputResources(ctx context.Context, secretTemplate *sg2v1alpha1.SecretTemplate) (map[string]unstructured.Unstructured, error) {
	unstructuredInputResources := map[string]unstructured.Unstructured{}

	for _, inputResource := range secretTemplate.Spec.InputResources {
		//Resolve resource
		inputResourceNamespace := secretTemplate.Namespace
		unstructuredResource, err := resolveInputResource(inputResource.Ref, inputResourceNamespace, unstructuredInputResources)
		if err != nil {
			return nil, err
		}

		key := types.NamespacedName{Namespace: inputResourceNamespace, Name: unstructuredResource.GetName()}

		//TODO: Setup dynamic watch

		//Fetch
		if err := r.client.Get(ctx, key, &unstructuredResource); err != nil {
			return nil, err
		}

		unstructuredInputResources[inputResource.Name] = unstructuredResource
	}
	return unstructuredInputResources, nil
}

func resolveInputResource(ref sg2v1alpha1.InputResourceRef, namespace string, inputResources map[string]unstructured.Unstructured) (unstructured.Unstructured, error) {
	//TODO: Resolve input resource from jsonpath templated inputResources

	//TODO resolve the name if it contains a jsonpath.
	//TODO check if jsonPath just returns string if no expression found.
	resolvedName, err := jsonPath(ref.Name, inputResources)
	if err != nil {
		return unstructured.Unstructured{}, err
	}

	return toUnstructured(ref.ApiVersion, ref.Kind, namespace, resolvedName.String())
}

func jsonPath(expression string, values interface{}) (*bytes.Buffer, error) {
	//TODO understand if we want allowmissingkeys or not.
	parser := jsonpath.New("").AllowMissingKeys(false)
	err := parser.Parse(expression)
	if err != nil {
		//todo template error
		return nil, err
	}

	buf := new(bytes.Buffer)
	err = parser.Execute(buf, values)
	if err != nil {
		//todo json path execute error
		return nil, err
	}

	return buf, nil
}

func toUnstructured(apiVersion, kind, namespace, name string) (unstructured.Unstructured, error) {
	gv, err := schema.ParseGroupVersion(apiVersion)
	if err != nil {
		return unstructured.Unstructured{}, err
	}

	gvk := schema.GroupVersionKind{
		Group:   gv.Group,
		Version: gv.Version,
		Kind:    kind,
	}

	obj := unstructured.Unstructured{}
	obj.SetGroupVersionKind(gvk)
	obj.SetName(name)
	obj.SetName(namespace)

	return obj, nil
}
