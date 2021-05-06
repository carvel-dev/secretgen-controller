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

type SecretExportReconciler struct {
	sgClient   sgclient.Interface
	coreClient kubernetes.Interface
	log        logr.Logger
}

var _ reconcile.Reconciler = &SecretExportReconciler{}

func NewSecretExportReconciler(sgClient sgclient.Interface,
	coreClient kubernetes.Interface, log logr.Logger) *SecretExportReconciler {
	return &SecretExportReconciler{sgClient, coreClient, log}
}

func (r *SecretExportReconciler) AttachWatches(controller controller.Controller) error {
	// Watch exported secrets and enqueue for same named SecretExports
	return controller.Watch(&source.Kind{Type: &corev1.Secret{}}, &handler.EnqueueRequestsFromMapFunc{
		ToRequests: handler.ToRequestsFunc(r.mapSecretToExport),
	})
}

func (r *SecretExportReconciler) mapSecretToExport(a handler.MapObject) []reconcile.Request {
	return []reconcile.Request{
		{NamespacedName: types.NamespacedName{
			Name:      a.Meta.GetName(),
			Namespace: a.Meta.GetNamespace(),
		}},
	}
}

func (r *SecretExportReconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	secretExport, err := r.sgClient.SecretgenV1alpha1().SecretExports(
		request.Namespace).Get(request.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			// Do not requeue as there is nothing to do when export is deleted
			return reconcile.Result{}, nil
		}
		// Requeue to try to fetch export again
		return reconcile.Result{Requeue: true}, err
	}

	if secretExport.DeletionTimestamp != nil {
		// Do not requeue as there is nothing to do
		return reconcile.Result{}, nil
	}

	status := &reconciler.Status{
		secretExport.Status.GenericStatus,
		func(st sgv1alpha1.GenericStatus) { secretExport.Status.GenericStatus = st },
	}

	status.SetReconciling(secretExport.ObjectMeta)
	defer r.updateStatus(secretExport)

	return status.WithReconcileCompleted(r.reconcile(secretExport))
}

func (r *SecretExportReconciler) reconcile(secretExport *sgv1alpha1.SecretExport) (reconcile.Result, error) {
	err := secretExport.Validate()
	if err != nil {
		// Do not requeue as there is nothing this controller can do until secret export is fixed
		return reconcile.Result{}, err
	}

	// Clear out observed resource version
	secretExport.Status.ObservedSecretResourceVersion = ""

	secret, err := r.coreClient.CoreV1().Secrets(
		secretExport.Namespace).Get(secretExport.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			// Do not requeue as there is nothing this controller can do until secret appears
			return reconcile.Result{}, fmt.Errorf("Missing exported secret")
		}
		// Requeue to try to fetch exported secret again
		return reconcile.Result{Requeue: true}, fmt.Errorf("Getting exported secret: %s", err)
	}

	// An update to export lets others know to reevaluate export
	secretExport.Status.ObservedSecretResourceVersion = secret.ResourceVersion

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
