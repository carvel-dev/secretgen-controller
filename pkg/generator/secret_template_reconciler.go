// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package generator

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

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
)

const (
	// Matches default sync in kapp-controller
	// See: https://github.com/vmware-tanzu/carvel-kapp-controller/blob/develop/pkg/app/reconcile_timer.go
	syncPeriod = 30 * time.Second
)

// SecretTemplateReconciler watches for SecretTemplate Resources and generates a new secret from a set of input resources.
type SecretTemplateReconciler struct {
	client   client.Client
	saLoader *ServiceAccountLoader
	log      logr.Logger
}

var _ reconcile.Reconciler = &SecretTemplateReconciler{}

// NewSecretTemplateReconciler create a new SecretTemplate Reconciler
func NewSecretTemplateReconciler(client client.Client, loader *ServiceAccountLoader, log logr.Logger) *SecretTemplateReconciler {
	return &SecretTemplateReconciler{client, loader, log}
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

	res, err := r.reconcile(ctx, &secretTemplate)
	//TODO is this overly defensive?
	if err != nil {
		if deleteErr := r.deleteChildSecret(ctx, &secretTemplate); deleteErr != nil {
			return status.WithReconcileCompleted(res, deleteErr)
		}
	}

	return status.WithReconcileCompleted(res, err)
}

func (r *SecretTemplateReconciler) reconcile(ctx context.Context, secretTemplate *sg2v1alpha1.SecretTemplate) (reconcile.Result, error) {

	//Get client to fetch inputResources
	inputResourceclient, err := r.clientForSecretTemplate(ctx, secretTemplate)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("unable to load client for reading Input Resources: %w", err)
	}

	//Resolve input resources
	inputResources, err := resolveInputResources(ctx, secretTemplate, inputResourceclient)
	if err != nil {
		return reconcile.Result{}, err
	}

	//Template Secret Data
	secretData := map[string][]byte{}
	for key, expression := range secretTemplate.Spec.JSONPathTemplate.Data {
		valueBuffer, err := JSONPath(expression).EvaluateWith(inputResources)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("templating data: %w", err)
		}

		decoded, err := base64.StdEncoding.DecodeString(valueBuffer.String())
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("failed decoding base64 from a Secret: %w", err)
		}

		secretData[key] = decoded
	}

	//Template Secret StringData
	secretStringData := map[string]string{}
	for key, expression := range secretTemplate.Spec.JSONPathTemplate.StringData {
		valueBuffer, err := JSONPath(expression).EvaluateWith(inputResources)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("templating stringData: %w", err)
		}

		secretStringData[key] = valueBuffer.String()
	}

	//Create Secret
	secret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretTemplate.GetName(),
			Namespace: secretTemplate.GetNamespace(),
		},
	}

	if _, err = controllerutil.CreateOrUpdate(ctx, r.client, &secret, func() error {
		secret.ObjectMeta.Labels = secretTemplate.GetLabels()
		secret.ObjectMeta.Annotations = secretTemplate.GetAnnotations()
		secret.StringData = secretStringData
		secret.Data = secretData

		// Secret Type is immutable, cannot update. TODO what is the best here?
		if secret.Type == "" {
			secret.Type = secretTemplate.Spec.JSONPathTemplate.Type
		}

		return controllerutil.SetControllerReference(secretTemplate, &secret, scheme.Scheme)
	}); err != nil {
		return reconcile.Result{}, fmt.Errorf("creating/updating secret: %w", err)
	}

	secretTemplate.Status.Secret.Name = secret.Name

	return reconcile.Result{
		RequeueAfter: syncPeriod,
	}, nil
}

func (r *SecretTemplateReconciler) updateStatus(ctx context.Context, secretTemplate *sg2v1alpha1.SecretTemplate) error {
	existingSecretTemplate := sg2v1alpha1.SecretTemplate{}
	if err := r.client.Get(ctx, types.NamespacedName{Namespace: secretTemplate.Namespace, Name: secretTemplate.Name}, &existingSecretTemplate); err != nil {
		return fmt.Errorf("fetching secretTemplate: %w", err)
	}

	existingSecretTemplate.Status = secretTemplate.Status

	if err := r.client.Status().Update(ctx, &existingSecretTemplate); err != nil {
		return fmt.Errorf("updating secretTemplate status: %w", err)
	}

	return nil
}

func (r *SecretTemplateReconciler) deleteChildSecret(ctx context.Context, secretTemplate *sg2v1alpha1.SecretTemplate) error {
	secret := corev1.Secret{}
	if err := r.client.Get(ctx, types.NamespacedName{Namespace: secretTemplate.GetName(), Name: secretTemplate.GetNamespace()}, &secret); err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
	}

	if err := r.client.Delete(ctx, &secret); err != nil {
		return fmt.Errorf("deleting secret: %w", err)
	}

	return nil
}

// Returns a client that was created using Service Account specified in the SecretTemplate spec.
// If no service account was specified then it returns the same Client as used by the SecretTemplateReconciler.
func (r *SecretTemplateReconciler) clientForSecretTemplate(ctx context.Context, secretTemplate *sg2v1alpha1.SecretTemplate) (client.Client, error) {
	c := r.client
	if secretTemplate.Spec.ServiceAccountName != "" {
		saClient, err := r.saLoader.Client(ctx, secretTemplate.Spec.ServiceAccountName, secretTemplate.Namespace)
		if err != nil {
			return nil, err
		}
		c = saClient
	}
	return c, nil
}

func resolveInputResources(ctx context.Context, secretTemplate *sg2v1alpha1.SecretTemplate, client client.Client) (map[string]interface{}, error) {
	resolvedInputResources := map[string]interface{}{}

	for _, inputResource := range secretTemplate.Spec.InputResources {
		unstructuredResource, err := resolveInputResource(inputResource.Ref, secretTemplate.Namespace, resolvedInputResources)
		if err != nil {
			return nil, fmt.Errorf("unable to resolve input resource %s: %w", inputResource.Name, err)
		}

		key := types.NamespacedName{Namespace: secretTemplate.Namespace, Name: unstructuredResource.GetName()}

		//TODO: Setup dynamic watch - first pass periodically re-reconciles
		if err := client.Get(ctx, key, &unstructuredResource); err != nil {
			return nil, fmt.Errorf("cannot fetch input resource %s: %w", unstructuredResource.GetName(), err)
		}

		resolvedInputResources[inputResource.Name] = unstructuredResource.UnstructuredContent()
	}
	return resolvedInputResources, nil
}

func resolveInputResource(ref sg2v1alpha1.InputResourceRef, namespace string, inputResources map[string]interface{}) (unstructured.Unstructured, error) {
	// Only support jsonpath for Input Resource Reference Names.
	resolvedName, err := JSONPath(ref.Name).EvaluateWith(inputResources)
	if err != nil {
		return unstructured.Unstructured{}, err
	}

	return toUnstructured(ref.APIVersion, ref.Kind, namespace, resolvedName.String())
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
