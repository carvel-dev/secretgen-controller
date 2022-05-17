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

type SecretTemplateReconciler struct {
	client   client.Client
	saLoader *ServiceAccountLoader
	log      logr.Logger
}

var _ reconcile.Reconciler = &SecretTemplateReconciler{}

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

	defer r.updateStatus(ctx, &secretTemplate)
	res, err := r.reconcile(ctx, &secretTemplate)

	if err != nil {
		if deleteErr := r.deleteChildSecret(ctx, &secretTemplate); deleteErr != nil {
			return reconcile.Result{}, secretTemplate.Status.WithReady(deleteErr)
		}
	}

	return res, secretTemplate.Status.WithReady(err)
}

func (r *SecretTemplateReconciler) reconcile(ctx context.Context, secretTemplate *sg2v1alpha1.SecretTemplate) (reconcile.Result, error) {
	secretTemplate.Status.InitializeConditions()

	//Get client to fetch inputResources
	inputResourceclient, err := r.clientForSecretTemplate(ctx, secretTemplate)
	if err != nil {
		return reconcile.Result{}, err
	}

	//Resolve input resources
	inputResources, err := resolveInputResources(ctx, secretTemplate, inputResourceclient)
	if err != nil {
		secretTemplate.Status.UpdateCondition(sgv1alpha1.Condition{
			Type:    sg2v1alpha1.InputResourcesFound,
			Status:  corev1.ConditionFalse,
			Reason:  "UnableToResolveInputResources",
			Message: err.Error(),
		})
		return reconcile.Result{}, err
	}

	secretTemplate.Status.UpdateCondition(sgv1alpha1.Condition{
		Type:   sg2v1alpha1.InputResourcesFound,
		Status: corev1.ConditionTrue,
	})

	//Template Secret Data
	secretData := map[string][]byte{}
	for key, expression := range secretTemplate.Spec.JSONPathTemplate.Data {
		valueBuffer, err := JSONPath(expression).EvaluateWith(inputResources)
		if err != nil {
			dataErr := fmt.Errorf("templating data: %w", err)
			secretTemplate.Status.UpdateCondition(sgv1alpha1.Condition{
				Type:    sg2v1alpha1.TemplatingSucceeded,
				Status:  corev1.ConditionFalse,
				Reason:  "UnableToTemplateSecretData",
				Message: dataErr.Error(),
			})
			return reconcile.Result{}, dataErr
		}

		decoded, err := base64.StdEncoding.DecodeString(valueBuffer.String())
		if err != nil {
			panic("should not get here as we are decoding base64 from a Secret")
		}

		secretData[key] = decoded
	}

	//Template Secret StringData
	secretStringData := map[string]string{}
	for key, expression := range secretTemplate.Spec.JSONPathTemplate.StringData {
		valueBuffer, err := JSONPath(expression).EvaluateWith(inputResources)
		if err != nil {
			stringDataErr := fmt.Errorf("templating stringData: %w", err)
			secretTemplate.Status.UpdateCondition(sgv1alpha1.Condition{
				Type:    sg2v1alpha1.TemplatingSucceeded,
				Status:  corev1.ConditionFalse,
				Reason:  "UnableToTemplateSecretStringData",
				Message: stringDataErr.Error(),
			})

			return reconcile.Result{}, stringDataErr
		}

		secretStringData[key] = valueBuffer.String()
	}

	secretTemplate.Status.UpdateCondition(sgv1alpha1.Condition{
		Type:   sg2v1alpha1.TemplatingSucceeded,
		Status: corev1.ConditionTrue,
	})

	//Create Secret
	secret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretTemplate.GetName(),
			Namespace: secretTemplate.GetNamespace(),
		},
	}

	if op, err := controllerutil.CreateOrUpdate(ctx, r.client, &secret, func() error {
		secret.ObjectMeta.Labels = secretTemplate.GetLabels()
		secret.ObjectMeta.Annotations = secretTemplate.GetAnnotations()
		secret.StringData = secretStringData
		secret.Data = secretData

		return controllerutil.SetControllerReference(secretTemplate, &secret, scheme.Scheme)
	}); err != nil {
		var reason string
		switch op {
		case controllerutil.OperationResultUpdated:
			reason = "UnableToUpdateSecret"
		case controllerutil.OperationResultCreated:
			reason = "UnableToCreateSecret"
		}
		secretTemplate.Status.UpdateCondition(sgv1alpha1.Condition{
			Type:    sg2v1alpha1.SecretCreated,
			Status:  corev1.ConditionFalse,
			Reason:  reason,
			Message: err.Error(),
		})

		return reconcile.Result{}, err
	}

	secretTemplate.Status.Secret.Name = secret.Name

	secretTemplate.Status.UpdateCondition(sgv1alpha1.Condition{
		Type:   sg2v1alpha1.SecretCreated,
		Status: corev1.ConditionTrue,
	})

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
			return nil, err
		}

		key := types.NamespacedName{Namespace: secretTemplate.Namespace, Name: unstructuredResource.GetName()}

		//TODO: Setup dynamic watch - first pass periodically re-reconciles
		if err := client.Get(ctx, key, &unstructuredResource); err != nil {
			return nil, err
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
