package reconciler

import (
	"fmt"

	"github.com/go-logr/logr"
	sgv1alpha1 "github.com/k14s/secretgen-controller/pkg/apis/secretgen/v1alpha1"
	sgclient "github.com/k14s/secretgen-controller/pkg/client/clientset/versioned"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

type SecretExportApprovalReconciler struct {
	sgClient   sgclient.Interface
	coreClient kubernetes.Interface
	log        logr.Logger
}

var _ reconcile.Reconciler = &SecretExportApprovalReconciler{}

func NewSecretExportApprovalReconciler(sgClient sgclient.Interface,
	coreClient kubernetes.Interface, log logr.Logger) *SecretExportApprovalReconciler {
	return &SecretExportApprovalReconciler{sgClient, coreClient, log}
}

func (r *SecretExportApprovalReconciler) AttachWatches(controller controller.Controller) error {
	var errs []error

	// Watch secrets and enqueue for same named SecretExportApproval
	// to make sure imported secret is up-to-date
	errs = append(errs, controller.Watch(&source.Kind{Type: &corev1.Secret{}}, &handler.EnqueueRequestsFromMapFunc{
		ToRequests: handler.ToRequestsFunc(r.mapSecretToApproval),
	}))

	// Watch SecretExport and enqueue for related SecretExportApproval
	// based on export namespace configuration
	errs = append(errs, controller.Watch(&source.Kind{Type: &sgv1alpha1.SecretExport{}}, &handler.EnqueueRequestsFromMapFunc{
		ToRequests: handler.ToRequestsFunc(r.mapExportsToApprovals),
	}))

	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *SecretExportApprovalReconciler) mapSecretToApproval(a handler.MapObject) []reconcile.Request {
	return []reconcile.Request{
		{NamespacedName: types.NamespacedName{
			Name:      a.Meta.GetName(),
			Namespace: a.Meta.GetNamespace(),
		}},
	}
}

func (r *SecretExportApprovalReconciler) mapExportsToApprovals(a handler.MapObject) []reconcile.Request {
	var export sgv1alpha1.SecretExport
	var result []reconcile.Request

	err := scheme.Scheme.Convert(a.Object, &export, nil)
	if err != nil {
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

func (r *SecretExportApprovalReconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	secretExportApproval, err := r.sgClient.SecretgenV1alpha1().SecretExportApprovals(
		request.Namespace).Get(request.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			// Do not requeue as there is nothing to do when approval is deleted
			return reconcile.Result{}, nil
		}
		// Requeue to try to fetch approval again
		return reconcile.Result{Requeue: true}, err
	}

	if secretExportApproval.DeletionTimestamp != nil {
		// Do not requeue as there is nothing to do
		// Associated secret has owned ref so it's going to be deleted
		return reconcile.Result{}, nil
	}

	status := &Status{
		secretExportApproval.Status.GenericStatus,
		func(st sgv1alpha1.GenericStatus) { secretExportApproval.Status.GenericStatus = st },
	}

	status.SetReconciling(secretExportApproval.ObjectMeta)
	defer r.updateStatus(secretExportApproval)

	return status.WithReconcileCompleted(r.reconcile(secretExportApproval))
}

func (r *SecretExportApprovalReconciler) reconcile(
	exportApproval *sgv1alpha1.SecretExportApproval) (reconcile.Result, error) {

	notOfferedMsg := "Export was not offered (even though approved)"
	notAllowedMsg := "Export was not allowed (even though approved)"

	secretExport, err := r.sgClient.SecretgenV1alpha1().SecretExports(
		exportApproval.Spec.FromNamespace).Get(exportApproval.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			// TODO Should we actually delete offered secret that we previously created?
			err := r.deleteAssociatedSecret(exportApproval)
			if err != nil {
				// Requeue to try to delete a bit later
				return reconcile.Result{Requeue: true}, fmt.Errorf("%s: %s", notOfferedMsg, err)
			}
			// Do not requeue since export is not offered
			return reconcile.Result{}, fmt.Errorf("%s", notOfferedMsg)
		}
		// Requeue to try to find secret export
		return reconcile.Result{Requeue: true}, fmt.Errorf("Finding export: %s", err)
	}

	if !r.isExportAllowed(secretExport, exportApproval) {
		err := r.deleteAssociatedSecret(exportApproval)
		if err != nil {
			// Requeue to try to delete a bit later
			return reconcile.Result{Requeue: true}, fmt.Errorf("%s: %s", notAllowedMsg, err)
		}
		// Do not requeue since export is not allowed
		return reconcile.Result{}, fmt.Errorf("%s", notAllowedMsg)
	}

	return r.copyAssociatedSecret(exportApproval)
}

func (r *SecretExportApprovalReconciler) isExportAllowed(
	export *sgv1alpha1.SecretExport, exportApproval *sgv1alpha1.SecretExportApproval) bool {

	if export.Spec.ToNamespace == exportApproval.Namespace {
		return true
	}
	for _, exportNs := range export.Spec.ToNamespaces {
		if exportNs == exportApproval.Namespace {
			return true
		}
	}
	return false
}

func (r *SecretExportApprovalReconciler) copyAssociatedSecret(
	exportApproval *sgv1alpha1.SecretExportApproval) (reconcile.Result, error) {

	srcSecret, err := r.coreClient.CoreV1().Secrets(
		exportApproval.Spec.FromNamespace).Get(exportApproval.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			// TODO Should we actually delete offered secret that we previously created?
			err := r.deleteAssociatedSecret(exportApproval)
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

	secret := NewSecret(exportApproval, nil)
	secret.ApplySecret(srcSecret)

	_, err = r.coreClient.CoreV1().Secrets(exportApproval.Namespace).Create(secret.AsSecret())
	switch {
	case err == nil:
		// Do not requeue since we copied secret successfully
		return reconcile.Result{}, nil

	case errors.IsAlreadyExists(err):
		existingSecret, err := r.coreClient.CoreV1().Secrets(exportApproval.Namespace).Get(
			exportApproval.Name, metav1.GetOptions{})
		if err != nil {
			// Requeue to try to fetch a bit later
			return reconcile.Result{Requeue: true}, fmt.Errorf("Getting imported secret: %s", err)
		}

		secret.AssociteExistingSecret(existingSecret)

		_, err = r.coreClient.CoreV1().Secrets(exportApproval.Namespace).Update(secret.AsSecret())
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

func (r *SecretExportApprovalReconciler) deleteAssociatedSecret(
	exportApproval *sgv1alpha1.SecretExportApproval) error {

	err := r.coreClient.CoreV1().Secrets(exportApproval.Namespace).Delete(
		exportApproval.Name, &metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("Deleting associated secret: %s", err)
	}
	return nil
}

func (r *SecretExportApprovalReconciler) updateStatus(
	exportApproval *sgv1alpha1.SecretExportApproval) error {

	existingSecretExportApproval, err := r.sgClient.SecretgenV1alpha1().SecretExportApprovals(
		exportApproval.Namespace).Get(exportApproval.Name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("Fetching secret export: %s", err)
	}

	existingSecretExportApproval.Status = exportApproval.Status

	_, err = r.sgClient.SecretgenV1alpha1().SecretExportApprovals(
		exportApproval.Namespace).UpdateStatus(exportApproval)
	if err != nil {
		return fmt.Errorf("Updating secret export status: %s", err)
	}

	return nil
}
