package reconciler

import (
	"fmt"

	cfgtypes "github.com/cloudfoundry/config-server/types"
	"github.com/go-logr/logr"
	sgv1alpha1 "github.com/vmware-tanzu/carvel-secretgen-controller/pkg/apis/secretgen/v1alpha1"
	sgclient "github.com/vmware-tanzu/carvel-secretgen-controller/pkg/client/clientset/versioned"
	"github.com/vmware-tanzu/carvel-secretgen-controller/pkg/expansion"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type CertificateReconciler struct {
	sgClient   sgclient.Interface
	coreClient kubernetes.Interface
	log        logr.Logger
}

var _ reconcile.Reconciler = &CertificateReconciler{}

func NewCertificateReconciler(sgClient sgclient.Interface,
	coreClient kubernetes.Interface, log logr.Logger) *CertificateReconciler {
	return &CertificateReconciler{sgClient, coreClient, log}
}

func (r *CertificateReconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	log := r.log.WithValues("request", request)

	cert, err := r.sgClient.SecretgenV1alpha1().Certificates(request.Namespace).Get(request.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("Not found")
			return reconcile.Result{}, nil
		}
		return reconcile.Result{Requeue: true}, err
	}

	if cert.DeletionTimestamp != nil {
		// Nothing to do
		return reconcile.Result{}, nil
	}

	status := &Status{
		cert.Status.GenericStatus,
		func(st sgv1alpha1.GenericStatus) { cert.Status.GenericStatus = st },
	}

	status.SetReconciling(cert.ObjectMeta)
	defer r.updateStatus(cert)

	return status.WithReconcileCompleted(r.reconcile(cert))
}

func (r *CertificateReconciler) reconcile(cert *sgv1alpha1.Certificate) (reconcile.Result, error) {
	params := newCertParams(cert)

	existingSecret, err := r.coreClient.CoreV1().Secrets(cert.Namespace).Get(cert.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return r.createSecret(params, cert)
		}
		return reconcile.Result{Requeue: true}, err
	}

	if (GenerateInputs{params}).IsChanged(existingSecret.Annotations) {
		return r.updateSecret(params, cert, existingSecret)
	}

	return reconcile.Result{}, nil
}

func (r *CertificateReconciler) createSecret(params certParams, cert *sgv1alpha1.Certificate) (reconcile.Result, error) {
	certResult, err := r.generate(params, cert)
	if err != nil {
		return reconcile.Result{Requeue: true}, err
	}

	values := map[string][]byte{
		sgv1alpha1.CertificateSecretCertificateKey: []byte(certResult.Certificate),
		sgv1alpha1.CertificateSecretPrivateKeyKey:  []byte(certResult.PrivateKey),
	}

	secret := NewSecret(cert, values)

	defaultTemplate := sgv1alpha1.SecretTemplate{
		Type: sgv1alpha1.CertificateSecretDefaultType,
		StringData: map[string]string{
			sgv1alpha1.CertificateSecretDefaultCertificateKey: expansion.Variable(sgv1alpha1.CertificateSecretCertificateKey),
			sgv1alpha1.CertificateSecretDefaultPrivateKeyKey:  expansion.Variable(sgv1alpha1.CertificateSecretPrivateKeyKey),
		},
	}

	err = secret.ApplyTemplates(defaultTemplate, cert.Spec.SecretTemplate)
	if err != nil {
		return reconcile.Result{Requeue: true}, err
	}

	newSecret := secret.AsSecret()

	GenerateInputs{params}.Add(newSecret.Annotations)

	_, err = r.coreClient.CoreV1().Secrets(newSecret.Namespace).Create(newSecret)
	if err != nil {
		return reconcile.Result{Requeue: true}, err
	}

	return reconcile.Result{}, nil
}

func (r *CertificateReconciler) updateSecret(params certParams,
	cert *sgv1alpha1.Certificate, secret *corev1.Secret) (reconcile.Result, error) {

	// TODO implement

	return reconcile.Result{}, nil
}

type certParams struct {
	CommonName       string   `yaml:"common_name"`
	Organization     string   `yaml:"organization"`
	AlternativeNames []string `yaml:"alternative_names"`
	IsCA             bool     `yaml:"is_ca"`
	CAName           string   `yaml:"ca"`
	ExtKeyUsage      []string `yaml:"extended_key_usage"`
	Duration         int64    `yaml:"duration"`
}

func newCertParams(cert *sgv1alpha1.Certificate) certParams {
	params := certParams{
		CommonName:       cert.Spec.CommonName,
		Organization:     cert.Spec.Organization,
		AlternativeNames: cert.Spec.AlternativeNames,
		IsCA:             cert.Spec.IsCA,
		ExtKeyUsage:      cert.Spec.ExtendedKeyUsage,
		Duration:         cert.Spec.Duration,
	}

	if len(params.Organization) == 0 {
		params.Organization = "secretgen"
	}

	if cert.Spec.CARef != nil {
		// Since loader is built with only one cert,
		// name is not relevant, but needs to present
		params.CAName = "unused-but-not-empty"
	}

	return params
}

func (r *CertificateReconciler) generate(params certParams,
	cert *sgv1alpha1.Certificate) (cfgtypes.CertResponse, error) {

	caCertSecret, err := r.getCARefSecret(cert)
	if err != nil {
		return cfgtypes.CertResponse{}, err
	}

	gen := cfgtypes.NewCertificateGenerator(singleCertLoader{caCertSecret})

	certVal, err := gen.Generate(params)
	if err != nil {
		return cfgtypes.CertResponse{}, err
	}

	return certVal.(cfgtypes.CertResponse), nil
}

func (r *CertificateReconciler) getCARefSecret(
	cert *sgv1alpha1.Certificate) (*corev1.Secret, error) {

	if cert.Spec.CARef == nil {
		return nil, nil
	}

	caSecret, err := r.coreClient.CoreV1().Secrets(cert.Namespace).Get(cert.Spec.CARef.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return caSecret, nil
}

func (r *CertificateReconciler) updateStatus(cert *sgv1alpha1.Certificate) error {
	existingCert, err := r.sgClient.SecretgenV1alpha1().Certificates(cert.Namespace).Get(cert.Name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("Fetching cert: %s", err)
	}

	existingCert.Status = cert.Status

	_, err = r.sgClient.SecretgenV1alpha1().Certificates(existingCert.Namespace).UpdateStatus(existingCert)
	if err != nil {
		return fmt.Errorf("Updating cert status: %s", err)
	}

	return nil
}
