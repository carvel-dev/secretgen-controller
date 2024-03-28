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
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
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

// AttachWatches adds starts watches this reconciler requires.
func (r *RSAKeyReconciler) AttachWatches(controller controller.Controller) error {
	return controller.Watch(&source.Kind{Type: &sgv1alpha1.RSAKey{}}, &handler.EnqueueRequestForObject{})
}

// Reconcile is the entrypoint for incoming requests from k8s
func (r *RSAKeyReconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.WithValues("request", request)

	rsaKey, err := r.sgClient.SecretgenV1alpha1().RSAKeys(request.Namespace).Get(ctx, request.Name, metav1.GetOptions{})
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

	status := &reconciler.Status{
		rsaKey.Status.GenericStatus,
		func(st sgv1alpha1.GenericStatus) { rsaKey.Status.GenericStatus = st },
	}

	status.SetReconciling(rsaKey.ObjectMeta)
	defer r.updateStatus(ctx, rsaKey)

	return status.WithReconcileCompleted(r.reconcile(ctx, rsaKey))
}

func (r *RSAKeyReconciler) reconcile(ctx context.Context, rsaKey *sgv1alpha1.RSAKey) (reconcile.Result, error) {
	_, err := r.coreClient.CoreV1().Secrets(rsaKey.Namespace).Get(ctx, rsaKey.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return r.createSecret(ctx, rsaKey)
		}
		return reconcile.Result{Requeue: true}, err
	}
	return reconcile.Result{}, nil
}

func (r *RSAKeyReconciler) createSecret(ctx context.Context, rsaKey *sgv1alpha1.RSAKey) (reconcile.Result, error) {
	rsaKeyResult, err := r.generate(rsaKey)
	if err != nil {
		return reconcile.Result{Requeue: true}, err
	}

	values := map[string][]byte{
		sgv1alpha1.RSAKeySecretPublicKeyKey:  []byte(rsaKeyResult.PublicKey),
		sgv1alpha1.RSAKeySecretPrivateKeyKey: []byte(rsaKeyResult.PrivateKey),
	}

	secret := reconciler.NewSecret(rsaKey, values)

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

	_, err = r.coreClient.CoreV1().Secrets(newSecret.Namespace).Create(ctx, newSecret, metav1.CreateOptions{})
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

func (r *RSAKeyReconciler) updateStatus(ctx context.Context, rsaKey *sgv1alpha1.RSAKey) error {
	existingRSAKey, err := r.sgClient.SecretgenV1alpha1().RSAKeys(rsaKey.Namespace).Get(ctx, rsaKey.Name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("Fetching rsakey: %s", err)
	}

	existingRSAKey.Status = rsaKey.Status

	_, err = r.sgClient.SecretgenV1alpha1().RSAKeys(existingRSAKey.Namespace).UpdateStatus(ctx, existingRSAKey, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("Updating rsakey status: %s", err)
	}

	return nil
}
