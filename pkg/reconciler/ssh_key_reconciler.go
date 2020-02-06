package reconciler

import (
	"fmt"

	cfgtypes "github.com/cloudfoundry/config-server/types"
	"github.com/go-logr/logr"
	sgv1alpha1 "github.com/k14s/secretgen-controller/pkg/apis/secretgen/v1alpha1"
	sgclient "github.com/k14s/secretgen-controller/pkg/client/clientset/versioned"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type SSHKeyReconciler struct {
	sgClient   sgclient.Interface
	coreClient kubernetes.Interface
	log        logr.Logger
}

var _ reconcile.Reconciler = &SSHKeyReconciler{}

func NewSSHKeyReconciler(sgClient sgclient.Interface,
	coreClient kubernetes.Interface, log logr.Logger) *SSHKeyReconciler {
	return &SSHKeyReconciler{sgClient, coreClient, log}
}

func (r *SSHKeyReconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	log := r.log.WithValues("request", request)

	sshKey, err := r.sgClient.SecretgenV1alpha1().SSHKeys(request.Namespace).Get(request.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("Not found")
			return reconcile.Result{}, nil
		}
		return reconcile.Result{Requeue: true}, err
	}

	if sshKey.DeletionTimestamp != nil {
		// Nothing to do
		return reconcile.Result{}, nil
	}

	status := &Status{
		sshKey.Status.GenericStatus,
		func(st sgv1alpha1.GenericStatus) { sshKey.Status.GenericStatus = st },
	}

	status.SetReconciling(sshKey.ObjectMeta)
	defer r.updateStatus(sshKey)

	return status.WithReconcileCompleted(r.reconcile(sshKey))
}

func (r *SSHKeyReconciler) reconcile(sshKey *sgv1alpha1.SSHKey) (reconcile.Result, error) {
	_, err := r.coreClient.CoreV1().Secrets(sshKey.Namespace).Get(sshKey.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return r.createSecret(sshKey)
		}
		return reconcile.Result{Requeue: true}, err
	}
	return reconcile.Result{}, nil
}

func (r *SSHKeyReconciler) createSecret(sshKey *sgv1alpha1.SSHKey) (reconcile.Result, error) {
	sshKeyResult, err := r.generate(sshKey)
	if err != nil {
		return reconcile.Result{Requeue: true}, err
	}

	values := map[string][]byte{
		sgv1alpha1.SSHKeySecretPrivateKeyKey:    []byte(sshKeyResult.PrivateKey),
		sgv1alpha1.SSHKeySecretAuthorizedKeyKey: []byte(sshKeyResult.PublicKey),
	}

	secret := NewSecret(sshKey, values)

	defaultTemplate := sgv1alpha1.SecretTemplate{
		Type: sgv1alpha1.SSHKeySecretDefaultType,
		Data: map[string]string{
			sgv1alpha1.SSHKeySecretDefaultPrivateKeyKey:    sgv1alpha1.SSHKeySecretPrivateKeyKey,
			sgv1alpha1.SSHKeySecretDefaultAuthorizedKeyKey: sgv1alpha1.SSHKeySecretAuthorizedKeyKey,
		},
	}

	err = secret.ApplyTemplates(defaultTemplate, sshKey.Spec.SecretTemplate)
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

func (r *SSHKeyReconciler) generate(sshKey *sgv1alpha1.SSHKey) (cfgtypes.SSHKey, error) {
	gen := cfgtypes.NewSSHKeyGenerator()

	// TODO allow type and number of bits?
	sshKeyVal, err := gen.Generate(nil)
	if err != nil {
		return cfgtypes.SSHKey{}, err
	}

	return sshKeyVal.(cfgtypes.SSHKey), nil
}

func (r *SSHKeyReconciler) updateStatus(sshKey *sgv1alpha1.SSHKey) error {
	existingSSHKey, err := r.sgClient.SecretgenV1alpha1().SSHKeys(sshKey.Namespace).Get(sshKey.Name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("Fetching sshkey: %s", err)
	}

	existingSSHKey.Status = sshKey.Status

	_, err = r.sgClient.SecretgenV1alpha1().SSHKeys(existingSSHKey.Namespace).UpdateStatus(existingSSHKey)
	if err != nil {
		return fmt.Errorf("Updating sshkey status: %s", err)
	}

	return nil
}
