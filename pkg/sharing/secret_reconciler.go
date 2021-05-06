package sharing

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"time"

	"github.com/go-logr/logr"
	sgv1alpha1 "github.com/vmware-tanzu/carvel-secretgen-controller/pkg/apis/secretgen/v1alpha1"
	sgclient "github.com/vmware-tanzu/carvel-secretgen-controller/pkg/client/clientset/versioned"
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

type SecretReconciler struct {
	sgClient      sgclient.Interface
	coreClient    kubernetes.Interface
	secretExports *SecretExports
	log           logr.Logger
}

var _ reconcile.Reconciler = &SecretReconciler{}

func NewSecretReconciler(sgClient sgclient.Interface, coreClient kubernetes.Interface,
	secretExports *SecretExports, log logr.Logger) *SecretReconciler {
	return &SecretReconciler{sgClient, coreClient, secretExports, log}
}

func (r *SecretReconciler) AttachWatches(controller controller.Controller) error {
	err := controller.Watch(&source.Kind{Type: &corev1.Secret{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return fmt.Errorf("Watching secrets: %s", err)
	}

	return controller.Watch(&source.Kind{Type: &sgv1alpha1.SecretExport{}}, &handler.EnqueueRequestsFromMapFunc{
		ToRequests: handler.ToRequestsFunc(r.mapSecretExportToSecret),
	})
}

func (r *SecretReconciler) mapSecretExportToSecret(a handler.MapObject) []reconcile.Request {
	// TODO expensive call on every secret export update (no cached client used, etc)
	secretList, err := r.coreClient.CoreV1().Secrets("").List(metav1.ListOptions{})
	if err != nil {
		r.log.Error(err, "Failed fetching list of all secrets")
		// TODO what should we really do here?
		return nil
	}

	var result []reconcile.Request
	for _, secret := range secretList.Items {
		result = append(result, reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      secret.Name,
				Namespace: secret.Namespace,
			},
		})
	}
	return result
}

func (r *SecretReconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	log := r.log.WithValues("request", request)

	secret, err := r.coreClient.CoreV1().Secrets(request.Namespace).Get(request.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{Requeue: true}, err
	}

	if secret.DeletionTimestamp != nil {
		// Nothing to do
		return reconcile.Result{}, nil
	}

	return r.reconcile(secret, secret.DeepCopy(), log)
}

func (r *SecretReconciler) reconcile(secret, originalSecret *corev1.Secret, log logr.Logger) (reconcile.Result, error) {
	const (
		imagePullSecretAnnKey = "secretgen.carvel.dev/image-pull-secret"
	)

	if _, found := secret.Annotations[imagePullSecretAnnKey]; !found {
		return reconcile.Result{}, nil
	}

	log.Info("Reconciling")

	// Note that "type" is immutable on a secret
	if secret.Type != corev1.SecretTypeDockerConfigJson {
		status := SecretStatus{
			Conditions: []sgv1alpha1.Condition{{
				Type:    sgv1alpha1.ReconcileFailed,
				Status:  corev1.ConditionTrue,
				Message: "Expected secret to have corev1.SecretTypeDockerConfigJson but did not",
			}},
		}
		return r.updateSecret(secret, status, originalSecret)
	}

	matcher := SecretMatcher{Namespace: secret.Namespace, SecretType: secret.Type}
	secrets := r.secretExports.MatchedSecretsForImport(matcher)

	newData, err := NewCombinedDockerConfigJSON(secrets)
	if err != nil {
		return reconcile.Result{RequeueAfter: 3 * time.Second}, err
	}

	secret.Data = newData

	status := SecretStatus{
		Conditions: []sgv1alpha1.Condition{{
			Type:   sgv1alpha1.ReconcileSucceeded,
			Status: corev1.ConditionTrue,
		}},
		SecretNames: r.statusSecretNames(secrets),
	}

	return r.updateSecret(secret, status, originalSecret)
}

func (r *SecretReconciler) updateSecret(secret *corev1.Secret, status SecretStatus,
	originalSecret *corev1.Secret) (reconcile.Result, error) {

	const (
		statusFieldAnnKey = "secretgen.carvel.dev/status"
	)

	encodedStatus, err := json.Marshal(status)
	if err != nil {
		// Requeue to try to update a bit later
		return reconcile.Result{Requeue: true}, fmt.Errorf("Marshaling secret status: %s", err)
	}

	if secret.Annotations == nil {
		secret.Annotations = map[string]string{}
	}
	secret.Annotations[statusFieldAnnKey] = string(encodedStatus)

	if reflect.DeepEqual(secret, originalSecret) {
		// Nothing changed leave early
		return reconcile.Result{}, nil
	}

	// TODO bother to retry to avoid having to recalculate matched secrets?
	_, err = r.coreClient.CoreV1().Secrets(secret.Namespace).Update(secret)
	if err != nil {
		// Requeue to try to update a bit later
		return reconcile.Result{Requeue: true}, fmt.Errorf("Updating secret: %s", err)
	}

	return reconcile.Result{}, nil
}

type SecretStatus struct {
	Conditions  []sgv1alpha1.Condition `json:"conditions,omitempty"`
	SecretNames []string               `json:"secretNames,omitempty"`
}

func (*SecretReconciler) statusSecretNames(secrets []*corev1.Secret) []string {
	var result []string
	for _, secret := range secrets {
		result = append(result, secret.Namespace+"/"+secret.Name)
	}
	sort.Strings(result)
	return result
}
