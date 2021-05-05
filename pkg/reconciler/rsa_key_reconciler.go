package reconciler

import (
	"fmt"

	cfgtypes "github.com/cloudfoundry/config-server/types"
	"github.com/go-logr/logr"
	sgv1alpha1 "github.com/vmware-tanzu/carvel-secretgen-controller/pkg/apis/secretgen/v1alpha1"
	sgclient "github.com/vmware-tanzu/carvel-secretgen-controller/pkg/client/clientset/versioned"
	"github.com/vmware-tanzu/carvel-secretgen-controller/pkg/expansion"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type RSAKeyReconciler struct {
	sgClient   sgclient.Interface
	coreClient kubernetes.Interface
	log        logr.Logger
}

var _ reconcile.Reconciler = &RSAKeyReconciler{}

func NewRSAKeyReconciler(sgClient sgclient.Interface,
	coreClient kubernetes.Interface, log logr.Logger) *RSAKeyReconciler {
	return &RSAKeyReconciler{sgClient, coreClient, log}
}

func (r *RSAKeyReconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	log := r.log.WithValues("request", request)

	rsaKey, err := r.sgClient.SecretgenV1alpha1().RSAKeys(request.Namespace).Get(request.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("Not found")
			return reconcile.Result{}, nil
		}
		return reconcile.Result{Requeue: true}, err
	}

	if rsaKey.DeletionTimestamp != nil {
		// Nothing to do
		return reconcile.Result{}, nil
	}

	status := &Status{
		rsaKey.Status.GenericStatus,
		func(st sgv1alpha1.GenericStatus) { rsaKey.Status.GenericStatus = st },
	}

	status.SetReconciling(rsaKey.ObjectMeta)
	defer r.updateStatus(rsaKey)

	return status.WithReconcileCompleted(r.reconcile(rsaKey))
}

func (r *RSAKeyReconciler) reconcile(rsaKey *sgv1alpha1.RSAKey) (reconcile.Result, error) {
	_, err := r.coreClient.CoreV1().Secrets(rsaKey.Namespace).Get(rsaKey.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return r.createSecret(rsaKey)
		}
		return reconcile.Result{Requeue: true}, err
	}
	return reconcile.Result{}, nil
}

func (r *RSAKeyReconciler) createSecret(rsaKey *sgv1alpha1.RSAKey) (reconcile.Result, error) {
	rsaKeyResult, err := r.generate(rsaKey)
	if err != nil {
		return reconcile.Result{Requeue: true}, err
	}

	values := map[string][]byte{
		sgv1alpha1.RSAKeySecretPublicKeyKey:  []byte(rsaKeyResult.PublicKey),
		sgv1alpha1.RSAKeySecretPrivateKeyKey: []byte(rsaKeyResult.PrivateKey),
	}

	secret := NewSecret(rsaKey, values)

	defaultTemplate := sgv1alpha1.SecretTemplate{
		Type: sgv1alpha1.RSAKeySecretDefaultType,
		StringData: map[string]string{
			sgv1alpha1.RSAKeySecretDefaultPublicKeyKey:  expansion.Variable(sgv1alpha1.RSAKeySecretPublicKeyKey),
			sgv1alpha1.RSAKeySecretDefaultPrivateKeyKey: expansion.Variable(sgv1alpha1.RSAKeySecretPrivateKeyKey),
		},
	}

	err = secret.ApplyTemplates(defaultTemplate, rsaKey.Spec.SecretTemplate)
	if err != nil {
		return reconcile.Result{Requeue: true}, err
	}

	newSecret := secret.AsSecret()

	_, err = r.coreClient.CoreV1().Secrets(newSecret.Namespace).Create(newSecret)
	if err != nil {
		return reconcile.Result{Requeue: true}, err
	}

	return reconcile.Result{}, nil
}

func (r *RSAKeyReconciler) generate(rsaKey *sgv1alpha1.RSAKey) (cfgtypes.RSAKey, error) {
	gen := cfgtypes.NewRSAKeyGenerator()

	// TODO allow number of bits?
	rsaKeyVal, err := gen.Generate(nil)
	if err != nil {
		return cfgtypes.RSAKey{}, err
	}

	return rsaKeyVal.(cfgtypes.RSAKey), nil
}

func (r *RSAKeyReconciler) updateStatus(rsaKey *sgv1alpha1.RSAKey) error {
	existingRSAKey, err := r.sgClient.SecretgenV1alpha1().RSAKeys(rsaKey.Namespace).Get(rsaKey.Name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("Fetching rsakey: %s", err)
	}

	existingRSAKey.Status = rsaKey.Status

	_, err = r.sgClient.SecretgenV1alpha1().RSAKeys(existingRSAKey.Namespace).UpdateStatus(existingRSAKey)
	if err != nil {
		return fmt.Errorf("Updating rsakey status: %s", err)
	}

	return nil
}
