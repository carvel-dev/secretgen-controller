// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package generator

import (
	"context"
	"fmt"

	cfgtypes "github.com/cloudfoundry/config-server/types"
	"github.com/go-logr/logr"
	sgv1alpha1 "github.com/vmware-tanzu/carvel-secretgen-controller/pkg/apis/secretgen/v1alpha1"
	sgclient "github.com/vmware-tanzu/carvel-secretgen-controller/pkg/client/clientset/versioned"
	"github.com/vmware-tanzu/carvel-secretgen-controller/pkg/expansion"
	"github.com/vmware-tanzu/carvel-secretgen-controller/pkg/reconciler"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
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

// AttachWatches adds starts watches this reconciler requires.
func (r *CertificateReconciler) AttachWatches(controller controller.Controller) error {
	return controller.Watch(&source.Kind{Type: &sgv1alpha1.Certificate{}}, &handler.EnqueueRequestForObject{})
}

// Reconcile is the entrypoint for incoming requests from k8s
func (r *CertificateReconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.WithValues("request", request)

	cert, err := r.sgClient.SecretgenV1alpha1().Certificates(request.Namespace).Get(ctx, request.Name, metav1.GetOptions{})
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

	status := &reconciler.Status{
		cert.Status.GenericStatus,
		func(st sgv1alpha1.GenericStatus) { cert.Status.GenericStatus = st },
	}

	status.SetReconciling(cert.ObjectMeta)
	defer r.updateStatus(ctx, cert)

	return status.WithReconcileCompleted(r.reconcile(ctx, cert))
}

func (r *CertificateReconciler) reconcile(ctx context.Context, cert *sgv1alpha1.Certificate) (reconcile.Result, error) {
	params := newCertParams(cert)

	existingSecret, err := r.coreClient.CoreV1().Secrets(cert.Namespace).Get(ctx, cert.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return r.createSecret(ctx, params, cert)
		}
		return reconcile.Result{Requeue: true}, err
	}

	if (GenerateInputs{params}).IsChanged(existingSecret.Annotations) {
		return r.updateSecret(params, cert, existingSecret)
	}

	return reconcile.Result{}, nil
}

func (r *CertificateReconciler) createSecret(ctx context.Context, params certParams, cert *sgv1alpha1.Certificate) (reconcile.Result, error) {
	certResult, err := r.generate(ctx, params, cert)
	if err != nil {
		return reconcile.Result{Requeue: true}, err
	}

	values := map[string][]byte{
		sgv1alpha1.CertificateSecretCertificateKey: []byte(certResult.Certificate),
		sgv1alpha1.CertificateSecretPrivateKeyKey:  []byte(certResult.PrivateKey),
	}

	secret := reconciler.NewSecret(cert, values)

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

	if newSecret.Annotations == nil {
		newSecret.Annotations = map[string]string{}
	}
	err = GenerateInputs{params}.Add(newSecret.Annotations)
	if err != nil {
		return reconcile.Result{Requeue: true}, err
	}

	_, err = r.coreClient.CoreV1().Secrets(newSecret.Namespace).Create(ctx, newSecret, metav1.CreateOptions{})
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

func (r *CertificateReconciler) generate(ctx context.Context, params certParams,
	cert *sgv1alpha1.Certificate) (cfgtypes.CertResponse, error) {

	caCertSecret, err := r.getCARefSecret(ctx, cert)
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
	ctx context.Context, cert *sgv1alpha1.Certificate) (*corev1.Secret, error) {

	if cert.Spec.CARef == nil {
		return nil, nil
	}

	caSecret, err := r.coreClient.CoreV1().Secrets(cert.Namespace).Get(ctx, cert.Spec.CARef.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return caSecret, nil
}

func (r *CertificateReconciler) updateStatus(ctx context.Context, cert *sgv1alpha1.Certificate) error {
	existingCert, err := r.sgClient.SecretgenV1alpha1().Certificates(cert.Namespace).Get(ctx, cert.Name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("Fetching cert: %s", err)
	}

	existingCert.Status = cert.Status

	_, err = r.sgClient.SecretgenV1alpha1().Certificates(existingCert.Namespace).UpdateStatus(ctx, existingCert, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("Updating cert status: %s", err)
	}

	return nil
}
