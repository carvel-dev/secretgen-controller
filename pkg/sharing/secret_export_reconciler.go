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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// SecretExportReconciler watches SecretExport CRs to record which Secret resources are exported
// so that they could be imported in other namespaces.
type SecretExportReconciler struct {
	client        client.Client
	secretExports *SecretExports
	log           logr.Logger
	isWarmedUp    bool
}

var _ reconcile.Reconciler = &SecretExportReconciler{}

// NewSecretExportReconciler constructs SecretExportReconciler.
func NewSecretExportReconciler(client client.Client,
	secretExports *SecretExports, log logr.Logger) *SecretExportReconciler {
	return &SecretExportReconciler{client, secretExports, log, false}
}

func (r *SecretExportReconciler) AttachWatches(controller controller.Controller) error {
	err := controller.Watch(&source.Kind{Type: &sgv1alpha1.SecretExport{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return fmt.Errorf("Watching secret exports: %s", err)
	}

	// Watch exported secrets and enqueue for same named SecretExports
	return controller.Watch(&source.Kind{Type: &corev1.Secret{}}, handler.EnqueueRequestsFromMapFunc(func(a client.Object) []reconcile.Request {
		return []reconcile.Request{
			{NamespacedName: types.NamespacedName{
				Name:      a.GetName(),
				Namespace: a.GetNamespace(),
			}},
		}
	}))
}

// WarmUp hydrates SecretExports given to this SecretExportReconciler with latest
// secret exports. If this method is not called before using SecretExports then
// users of SecretExports such as SecretReconciler will not have complete/accurate data.
func (r *SecretExportReconciler) WarmUp(ctx context.Context) error {
	r.log.Info("Running WarmUp")
	defer func() { r.log.Info("Done running WarmUp") }()

	var secretExportList sgv1alpha1.SecretExportList

	err := r.client.List(ctx, &secretExportList)
	if err != nil {
		return err
	}

	r.log.Info("Warming up with N exports", "len", len(secretExportList.Items))

	for _, se := range secretExportList.Items {
		_, err := r.reconcile(ctx, &se, r.log)
		if err != nil {
			// Ignore error
		}
	}

	return nil
}

// Reconcile acs on a request for a SecretExport to implement a kubernetes reconciler
func (r *SecretExportReconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.WithValues("request", request)
	if !r.isWarmedUp {
		if err := r.WarmUp(ctx); err != nil {
			return reconcile.Result{}, nil // TODO : how should we handle a warmup failure here?
		}
		r.isWarmedUp = true
	}

	log.Info("Reconciling")

	var secretExport sgv1alpha1.SecretExport

	err := r.client.Get(ctx, request.NamespacedName, &secretExport)
	if err != nil {
		if errors.IsNotFound(err) {
			r.secretExports.Unexport(&sgv1alpha1.SecretExport{
				ObjectMeta: metav1.ObjectMeta{
					Name:      request.Name,
					Namespace: request.Namespace,
				},
			})
			// Do not requeue as there is nothing to do when export is deleted
			return reconcile.Result{}, nil
		}
		// Requeue to try to fetch export again
		return reconcile.Result{Requeue: true}, err
	}

	if secretExport.DeletionTimestamp != nil {
		r.secretExports.Unexport(&secretExport)
		// Do not requeue as there is nothing to do
		return reconcile.Result{}, nil
	}

	status := &reconciler.Status{
		secretExport.Status.GenericStatus,
		func(st sgv1alpha1.GenericStatus) { secretExport.Status.GenericStatus = st },
	}

	status.SetReconciling(secretExport.ObjectMeta)

	reconcileResult, reconcileErr := status.WithReconcileCompleted(r.reconcile(ctx, &secretExport, log))

	// Saving the status helps trigger a cascade so that
	// the Secrets reconciler will also respond if needed
	err = r.updateStatus(ctx, secretExport)
	if err != nil {
		return reconcile.Result{}, err
	}

	return reconcileResult, reconcileErr
}

// reconcile looks for the Secret corresponding to the SecretExport Request that we're reconciling.
func (r *SecretExportReconciler) reconcile(ctx context.Context, secretExport *sgv1alpha1.SecretExport, log logr.Logger) (reconcile.Result, error) {
	// Clear out observed resource version
	secretExport.Status.ObservedSecretResourceVersion = ""

	err := secretExport.Validate()
	if err != nil {
		// Drop the Secret from the shared cache.
		r.secretExports.Unexport(secretExport)
		// Do not requeue as there is nothing this controller can do until secret export is fixed
		return reconcile.Result{}, reconciler.TerminalReconcileErr{err}
	}

	var secret corev1.Secret
	secretNN := types.NamespacedName{Namespace: secretExport.Namespace, Name: secretExport.Name}

	err = r.client.Get(ctx, secretNN, &secret)
	if err != nil {
		if errors.IsNotFound(err) {
			// Drop the Secret from the shared cache.
			r.secretExports.Unexport(secretExport)
			// Do not requeue as there is nothing this controller can do until secret appears
			return reconcile.Result{}, reconciler.TerminalReconcileErr{fmt.Errorf("Missing exported secret")}
		}
		// Requeue to try to fetch exported secret again
		return reconcile.Result{Requeue: true}, fmt.Errorf("Getting exported secret: %s", err)
	}

	// An update to export lets others know to reevaluate export
	secretExport.Status.ObservedSecretResourceVersion = secret.ResourceVersion

	r.secretExports.Export(secretExport, &secret)

	// Do not requeue since we found exported secret
	return reconcile.Result{}, nil
}

func (r *SecretExportReconciler) updateStatus(ctx context.Context, secretExport sgv1alpha1.SecretExport) error {
	r.log.Info("Updating secret export")

	err := r.client.Status().Update(ctx, &secretExport)
	if err != nil {
		return fmt.Errorf("Updating secret export status: %s", err)
	}

	return nil
}
