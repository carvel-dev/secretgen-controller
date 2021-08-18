// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package sharing

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	sgv1alpha1 "github.com/vmware-tanzu/carvel-secretgen-controller/pkg/apis/secretgen/v1alpha1"
	"github.com/vmware-tanzu/carvel-secretgen-controller/pkg/reconciler"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	// metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// SecretRequestReconciler creates an imported Secret if it was exported.
type SecretRequestReconciler struct {
	client client.Client
	log    logr.Logger
}

var _ reconcile.Reconciler = &SecretRequestReconciler{}

func NewSecretRequestReconciler(client client.Client, log logr.Logger) *SecretRequestReconciler {
	return &SecretRequestReconciler{client, log}
}

func (r *SecretRequestReconciler) AttachWatches(controller controller.Controller) error {
	err := controller.Watch(&source.Kind{Type: &sgv1alpha1.SecretRequest{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return fmt.Errorf("Watching secret request: %s", err)
	}

	var errs []error

	// Watch secrets and enqueue for same named SecretRequest
	// to make sure imported secret is up-to-date
	errs = append(errs, controller.Watch(
		&source.Kind{Type: &corev1.Secret{}},
		&handler.EnqueueRequestsFromMapFunc{ToRequests: handler.ToRequestsFunc(r.mapSecretToRequest)},
	))

	// Watch SecretExport and enqueue for related SecretRequest
	// based on export namespace configuration
	errs = append(errs, controller.Watch(
		&source.Kind{Type: &sgv1alpha1.SecretExport{}},
		&handler.EnqueueRequestsFromMapFunc{ToRequests: handler.ToRequestsFunc(r.mapExportsToRequests)},
	))

	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *SecretRequestReconciler) mapSecretToRequest(a handler.MapObject) []reconcile.Request {
	return []reconcile.Request{
		{NamespacedName: types.NamespacedName{
			Name:      a.Meta.GetName(),
			Namespace: a.Meta.GetNamespace(),
		}},
	}
}

func (r *SecretRequestReconciler) mapExportsToRequests(a handler.MapObject) []reconcile.Request {
	var export sgv1alpha1.SecretExport
	var result []reconcile.Request

	err := scheme.Scheme.Convert(a.Object, &export, nil)
	if err != nil {
		return nil
	}

	// Skip exports that are not fully reconciled
	// New events will be emitted when reconciliation finishes
	if !(&reconciler.Status{S: export.Status.GenericStatus}).IsReconcileSucceeded() {
		return nil
	}

	for _, ns := range export.StaticToNamespaces() {
		result = append(result, reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      a.Meta.GetName(),
				Namespace: ns,
			},
		})
	}

	return result
}

func (r *SecretRequestReconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	log := r.log.WithValues("request", request)

	var secretRequest sgv1alpha1.SecretRequest

	err := r.client.Get(context.TODO(), request.NamespacedName, &secretRequest)
	if err != nil {
		if errors.IsNotFound(err) {
			// Do not requeue as there is nothing to do when request is deleted
			return reconcile.Result{}, nil
		}
		// Requeue to try to fetch request again
		return reconcile.Result{Requeue: true}, err
	}

	if secretRequest.DeletionTimestamp != nil {
		// Do not requeue as there is nothing to do
		// Associated secret has owned ref so it's going to be deleted
		return reconcile.Result{}, nil
	}

	status := &reconciler.Status{
		secretRequest.Status.GenericStatus,
		func(st sgv1alpha1.GenericStatus) { secretRequest.Status.GenericStatus = st },
	}

	status.SetReconciling(secretRequest.ObjectMeta)

	reconcileResult, reconcileErr := status.WithReconcileCompleted(r.reconcile(secretRequest, log))

	err = r.updateStatus(secretRequest)
	if err != nil {
		return reconcile.Result{}, err
	}

	return reconcileResult, reconcileErr
}

func (r *SecretRequestReconciler) reconcile(
	secretRequest sgv1alpha1.SecretRequest, log logr.Logger) (reconcile.Result, error) {

	err := secretRequest.Validate()
	if err != nil {
		// Do not requeue as there is nothing this controller can do until secret request is fixed
		return reconcile.Result{}, reconciler.TerminalReconcileErr{err}
	}

	log.Info("Reconciling")

	notOfferedMsg := "Export was not offered (even though requested)"
	notAllowedMsg := "Export was not allowed (even though requested)"

	var secretExport sgv1alpha1.SecretExport
	secretExportNN := types.NamespacedName{
		Namespace: secretRequest.Spec.FromNamespace,
		Name:      secretRequest.Name,
	}

	err = r.client.Get(context.TODO(), secretExportNN, &secretExport)
	if err != nil {
		if errors.IsNotFound(err) {
			// TODO Should we actually delete offered secret that we previously created?
			err := r.deleteAssociatedSecret(secretRequest)
			if err != nil {
				// Requeue to try to delete a bit later
				return reconcile.Result{Requeue: true}, fmt.Errorf("%s: %s", notOfferedMsg, err)
			}
			// Do not requeue since export is not offered
			return reconcile.Result{}, reconciler.TerminalReconcileErr{fmt.Errorf("%s", notOfferedMsg)}
		}
		// Requeue to try to find secret export
		return reconcile.Result{Requeue: true}, fmt.Errorf("Finding export: %s", err)
	}

	if !r.isExportAllowed(secretExport, secretRequest) {
		err := r.deleteAssociatedSecret(secretRequest)
		if err != nil {
			// Requeue to try to delete a bit later
			return reconcile.Result{Requeue: true}, err
		}
		// Do not requeue since export is not allowed
		return reconcile.Result{}, reconciler.TerminalReconcileErr{fmt.Errorf("%s", notAllowedMsg)}
	}

	return r.copyAssociatedSecret(secretRequest)
}

func (r *SecretRequestReconciler) isExportAllowed(
	export sgv1alpha1.SecretExport, secretRequest sgv1alpha1.SecretRequest) bool {

	if export.Spec.ToNamespace == secretRequest.Namespace {
		return true
	}
	for _, exportNs := range export.Spec.ToNamespaces {
		if exportNs == secretRequest.Namespace {
			return true
		}
	}
	return false
}

func (r *SecretRequestReconciler) copyAssociatedSecret(
	secretRequest sgv1alpha1.SecretRequest) (reconcile.Result, error) {

	var srcSecret corev1.Secret
	srcSecretNN := types.NamespacedName{
		Namespace: secretRequest.Spec.FromNamespace,
		Name:      secretRequest.Name,
	}

	err := r.client.Get(context.TODO(), srcSecretNN, &srcSecret)
	if err != nil {
		if errors.IsNotFound(err) {
			// TODO Should we actually delete offered secret that we previously created?
			err := r.deleteAssociatedSecret(secretRequest)
			if err != nil {
				// Requeue to try to delete a bit later
				return reconcile.Result{Requeue: true}, err
			}
			// Do not requeue since there is nothing this controller can do to fix that
			return reconcile.Result{}, nil
		}
		// Requeue to try to fetch a bit later
		return reconcile.Result{Requeue: true}, fmt.Errorf("Getting exported secret: %s", err)
	}

	secret := reconciler.NewSecret(&secretRequest, nil)
	secret.ApplySecret(srcSecret)

	err = r.client.Create(context.TODO(), secret.AsSecret())
	switch {
	case err == nil:
		// Do not requeue since we copied secret successfully
		return reconcile.Result{}, nil

	case errors.IsAlreadyExists(err):
		var existingSecret corev1.Secret
		existingSecretNN := types.NamespacedName{
			Namespace: secretRequest.Namespace,
			Name:      secretRequest.Name,
		}

		err := r.client.Get(context.TODO(), existingSecretNN, &existingSecret)
		if err != nil {
			// Requeue to try to fetch a bit later
			return reconcile.Result{Requeue: true}, fmt.Errorf("Getting imported secret: %s", err)
		}

		secret.AssociateExistingSecret(existingSecret)

		err = r.client.Update(context.TODO(), secret.AsSecret())
		if err != nil {
			// Requeue to try to update a bit later
			return reconcile.Result{Requeue: true}, fmt.Errorf("Updating imported secret: %s", err)
		}

		// Do not requeue since we copied secret successfully
		return reconcile.Result{}, nil

	default:
		// Requeue to try to create a bit later
		return reconcile.Result{Requeue: true}, fmt.Errorf("Creating imported secret: %s", err)
	}
}

func (r *SecretRequestReconciler) deleteAssociatedSecret(
	secretRequest sgv1alpha1.SecretRequest) error {

	var secret corev1.Secret
	secretNN := types.NamespacedName{
		Namespace: secretRequest.Namespace,
		Name:      secretRequest.Name,
	}

	// TODO get rid of extra get
	err := r.client.Get(context.TODO(), secretNN, &secret)
	if err != nil {
		return nil
	}

	err = r.client.Delete(context.TODO(), &secret)
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("Deleting associated secret: %s", err)
	}
	return nil
}

func (r *SecretRequestReconciler) updateStatus(
	secretRequest sgv1alpha1.SecretRequest) error {

	err := r.client.Status().Update(context.TODO(), &secretRequest)
	if err != nil {
		return fmt.Errorf("Updating secret request status: %s", err)
	}

	return nil
}
