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
	if err := controller.Watch(&source.Kind{Type: &corev1.Secret{}}, &handler.EnqueueRequestForOwner{OwnerType: &sg2v1alpha1.SecretTemplate{}}); err != nil {
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
		S:          secretTemplate.Status.GenericStatus,
		UpdateFunc: func(st sgv1alpha1.GenericStatus) { secretTemplate.Status.GenericStatus = st },
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

	//TODO handle existing secret when failing to fetch input resources.
	//When an input resource is deleted or a key is missing we should delete the secret.
	//But should we delete the secret for potentially intermitent errors?

	//Template Secret Data
	secretData := map[string][]byte{}
	for key, expression := range secretTemplate.Spec.JSONPathTemplate.Data {
		valueBuffer, err := jsonPath(expression, inputResources)
		if err != nil {
			//TODO jsonpath error
			//Delete any existing secret?
			return reconcile.Result{}, err
		}
		secretData[key] = valueBuffer.Bytes()
	}
	//Template Secret StringData
	secretStringData := map[string]string{}
	for key, expression := range secretTemplate.Spec.JSONPathTemplate.StringData {
		valueBuffer, err := jsonPath(expression, inputResources)
		if err != nil {
			//TODO jsonpath error
			//Delete any existing secret?
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

	if _, err := controllerutil.CreateOrUpdate(ctx, r.client, secret, func() error {
		secret.ObjectMeta.Labels = secretTemplate.GetLabels()           //TODO do we want these implicitly?
		secret.ObjectMeta.Annotations = secretTemplate.GetAnnotations() //TODO do we want these implicitly?
		secret.StringData = secretStringData
		secret.Data = secretData
		return nil
	}); err != nil {
		return reconcile.Result{}, err
	}

	secretTemplate.Status.CreatedSecret.Name = secret.Name

	return reconcile.Result{}, nil
}

func (r *SecretTemplateReconciler) updateStatus(ctx context.Context, secretTemplate *sg2v1alpha1.SecretTemplate) error {
	existingSecretTemplate := sg2v1alpha1.SecretTemplate{}
	if err := r.client.Get(ctx, types.NamespacedName{Namespace: secretTemplate.Namespace, Name: secretTemplate.Name}, &existingSecretTemplate); err != nil {
		return fmt.Errorf("fetching secretTemplate: %s", err)
	}

	existingSecretTemplate.Status = secretTemplate.Status

	if err := r.client.Status().Update(ctx, &existingSecretTemplate); err != nil {
		return fmt.Errorf("updating secretTemplate status: %s", err)
	}

	return nil
}

func (r *SecretTemplateReconciler) resolveInputResources(ctx context.Context, secretTemplate *sg2v1alpha1.SecretTemplate) (map[string]interface{}, error) {
	resolvedInputResources := map[string]interface{}{}

	for _, inputResource := range secretTemplate.Spec.InputResources {
		unstructuredResource, err := resolveInputResource(inputResource.Ref, secretTemplate.Namespace, resolvedInputResources)
		if err != nil {
			return nil, err
		}

		key := types.NamespacedName{Namespace: secretTemplate.Namespace, Name: unstructuredResource.GetName()}

		//TODO: Setup dynamic watch - maybe a first pass periodically re-reconciles (like kapp controller)

		//Fetch
		//TODO this should use a client from the Service Account - unless loading secrets(?)
		if err := r.client.Get(ctx, key, &unstructuredResource); err != nil {
			return nil, err
		}

		resolvedInputResources[inputResource.Name] = unstructuredResource.UnstructuredContent()
	}
	return resolvedInputResources, nil
}

func resolveInputResource(ref sg2v1alpha1.InputResourceRef, namespace string, inputResources map[string]interface{}) (unstructured.Unstructured, error) {
	//TODO should we only search for jsonpath expressions in name? Probably.
	resolvedName, err := jsonPath(ref.Name, inputResources)
	if err != nil {
		return unstructured.Unstructured{}, err
	}

	return toUnstructured(ref.APIVersion, ref.Kind, namespace, resolvedName.String())
}

//TODO how does this package from k8s align with our usecases? Do other packages exist?
// { .creds.data.inputKey1 } succeeds
// .creds.data.inputKey1 does not work
// We may have to extract jsonpath from our syntax e.g. `$( )` and then run execute this with { }.
func jsonPath(expression string, values interface{}) (*bytes.Buffer, error) {

	//TODO debugging, remove or log.
	fmt.Printf("jsonpath before ex: %s, values:%v\n", expression, values)

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

	//TODO debugging, remove or log.
	fmt.Printf("jsonpath result ex: %s, values:%v res:%s\n", expression, values, buf.String())

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
	obj.SetNamespace(namespace)

	return obj, nil
}
