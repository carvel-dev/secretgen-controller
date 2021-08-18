// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package sharing

import (
	"fmt"

	"github.com/go-logr/logr"
	sgv1alpha1 "github.com/vmware-tanzu/carvel-secretgen-controller/pkg/apis/secretgen/v1alpha1"
	sgclient "github.com/vmware-tanzu/carvel-secretgen-controller/pkg/client/clientset/versioned"
	"github.com/vmware-tanzu/carvel-secretgen-controller/pkg/reconciler"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// SecretExportReconciler watches SecretExport CRs to record which Secret resources are exported
// so that they could be imported in other namespaces.
type SecretExportReconciler struct {
	sgClient      sgclient.Interface
	coreClient    kubernetes.Interface
	secretExports *SecretExports
	log           logr.Logger
}

var _ reconcile.Reconciler = &SecretExportReconciler{}

func NewSecretExportReconciler(sgClient sgclient.Interface, coreClient kubernetes.Interface,
	secretExports *SecretExports, log logr.Logger) *SecretExportReconciler {
	return &SecretExportReconciler{sgClient, coreClient, secretExports, log}
}

func (r *SecretExportReconciler) AttachWatches(controller controller.Controller) error {
	err := controller.Watch(&source.Kind{Type: &sgv1alpha1.SecretExport{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return fmt.Errorf("Watching secret exports: %s", err)
	}

	// Watch exported secrets and enqueue for same named SecretExports
	return controller.Watch(&source.Kind{Type: &corev1.Secret{}}, &handler.EnqueueRequestsFromMapFunc{
		ToRequests: handler.ToRequestsFunc(func(a handler.MapObject) []reconcile.Request {
			return []reconcile.Request{
				{NamespacedName: types.NamespacedName{
					Name:      a.Meta.GetName(),
					Namespace: a.Meta.GetNamespace(),
				}},
			}
		}),
	})
}

// WarmUp hydrates SecretExports given to this SecretExportReconciler with latest
// secret exports. If this method is not called before using SecretExports then
// users of SecretExports such as SecretReconciler will not have complete/accurate data.
func (r *SecretExportReconciler) WarmUp() error {
	secretExportList, err := r.sgClient.SecretgenV1alpha1().SecretExports("").List(metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, se := range secretExportList.Items {
		_, err := r.reconcile(&se, r.log)
		if err != nil {
			// Ignore error
		}
	}

	return nil
}

// Reconcile acs on a request for a SecretExport to implement a kubernetes reconciler
func (r *SecretExportReconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	log := r.log.WithValues("request", request)

	secretExport, err := r.sgClient.SecretgenV1alpha1().SecretExports(
		request.Namespace).Get(request.Name, metav1.GetOptions{})
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
		r.secretExports.Unexport(secretExport)
		// Do not requeue as there is nothing to do
		return reconcile.Result{}, nil
	}

	status := &reconciler.Status{
		secretExport.Status.GenericStatus,
		func(st sgv1alpha1.GenericStatus) { secretExport.Status.GenericStatus = st },
	}

	status.SetReconciling(secretExport.ObjectMeta)
	// saving the status helps trigger a cascade so that the Secrets reconciler will also respond if needed
	defer r.updateStatus(secretExport)

	return status.WithReconcileCompleted(r.reconcile(secretExport, log))
}

// reconcile looks for the Secret corresponding to the SecretExport Request that we're reconciling.
func (r *SecretExportReconciler) reconcile(secretExport *sgv1alpha1.SecretExport, log logr.Logger) (reconcile.Result, error) {
	err := secretExport.Validate()
	if err != nil {
		// Do not requeue as there is nothing this controller can do until secret export is fixed
		return reconcile.Result{}, err
	}

	log.Info("Reconciling")

	// Clear out observed resource version
	secretExport.Status.ObservedSecretResourceVersion = ""

	secret, err := r.coreClient.CoreV1().Secrets(
		secretExport.Namespace).Get(secretExport.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			// drop the Secret from the shared cache.
			r.secretExports.Unexport(secretExport)
			// Do not requeue as there is nothing this controller can do until secret appears
			return reconcile.Result{}, fmt.Errorf("Missing exported secret")
		}
		// Requeue to try to fetch exported secret again
		return reconcile.Result{Requeue: true}, fmt.Errorf("Getting exported secret: %s", err)
	}

	// An update to export lets others know to reevaluate export
	secretExport.Status.ObservedSecretResourceVersion = secret.ResourceVersion

	r.secretExports.Export(secretExport, secret)

	// Do not requeue since we found exported secret
	return reconcile.Result{}, nil
}

func (r *SecretExportReconciler) updateStatus(secretExport *sgv1alpha1.SecretExport) error {
	existingSecretExport, err := r.sgClient.SecretgenV1alpha1().SecretExports(
		secretExport.Namespace).Get(secretExport.Name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("Fetching secret export: %s", err)
	}

	existingSecretExport.Status = secretExport.Status

	_, err = r.sgClient.SecretgenV1alpha1().SecretExports(
		existingSecretExport.Namespace).UpdateStatus(existingSecretExport)
	if err != nil {
		return fmt.Errorf("Updating secret export status: %s", err)
	}

	return nil
}
