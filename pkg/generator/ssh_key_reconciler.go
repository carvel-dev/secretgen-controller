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

// AttachWatches adds starts watches this reconciler requires.
func (r *SSHKeyReconciler) AttachWatches(controller controller.Controller) error {
	return controller.Watch(&source.Kind{Type: &sgv1alpha1.SSHKey{}}, &handler.EnqueueRequestForObject{})
}

// Reconcile is the entrypoint for incoming requests from k8s
func (r *SSHKeyReconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.WithValues("request", request)

	sshKey, err := r.sgClient.SecretgenV1alpha1().SSHKeys(request.Namespace).Get(ctx, request.Name, metav1.GetOptions{})
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

	status := &reconciler.Status{
		sshKey.Status.GenericStatus,
		func(st sgv1alpha1.GenericStatus) { sshKey.Status.GenericStatus = st },
	}

	status.SetReconciling(sshKey.ObjectMeta)
	defer r.updateStatus(ctx, sshKey)

	return status.WithReconcileCompleted(r.reconcile(ctx, sshKey))
}

func (r *SSHKeyReconciler) reconcile(ctx context.Context, sshKey *sgv1alpha1.SSHKey) (reconcile.Result, error) {
	_, err := r.coreClient.CoreV1().Secrets(sshKey.Namespace).Get(ctx, sshKey.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return r.createSecret(ctx, sshKey)
		}
		return reconcile.Result{Requeue: true}, err
	}
	return reconcile.Result{}, nil
}

func (r *SSHKeyReconciler) createSecret(ctx context.Context, sshKey *sgv1alpha1.SSHKey) (reconcile.Result, error) {
	sshKeyResult, err := r.generate(sshKey)
	if err != nil {
		return reconcile.Result{Requeue: true}, err
	}

	values := map[string][]byte{
		sgv1alpha1.SSHKeySecretPrivateKeyKey:    []byte(sshKeyResult.PrivateKey),
		sgv1alpha1.SSHKeySecretAuthorizedKeyKey: []byte(sshKeyResult.PublicKey),
	}

	secret := reconciler.NewSecret(sshKey, values)

	defaultTemplate := sgv1alpha1.SecretTemplate{
		Type: sgv1alpha1.SSHKeySecretDefaultType,
		StringData: map[string]string{
			sgv1alpha1.SSHKeySecretDefaultPrivateKeyKey:    expansion.Variable(sgv1alpha1.SSHKeySecretPrivateKeyKey),
			sgv1alpha1.SSHKeySecretDefaultAuthorizedKeyKey: expansion.Variable(sgv1alpha1.SSHKeySecretAuthorizedKeyKey),
		},
	}

	err = secret.ApplyTemplates(defaultTemplate, sshKey.Spec.SecretTemplate)
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

func (r *SSHKeyReconciler) generate(sshKey *sgv1alpha1.SSHKey) (cfgtypes.SSHKey, error) {
	gen := cfgtypes.NewSSHKeyGenerator()

	// TODO allow type and number of bits?
	sshKeyVal, err := gen.Generate(nil)
	if err != nil {
		return cfgtypes.SSHKey{}, err
	}

	return sshKeyVal.(cfgtypes.SSHKey), nil
}

func (r *SSHKeyReconciler) updateStatus(ctx context.Context, sshKey *sgv1alpha1.SSHKey) error {
	existingSSHKey, err := r.sgClient.SecretgenV1alpha1().SSHKeys(sshKey.Namespace).Get(ctx, sshKey.Name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("Fetching sshkey: %s", err)
	}

	existingSSHKey.Status = sshKey.Status

	_, err = r.sgClient.SecretgenV1alpha1().SSHKeys(existingSSHKey.Namespace).UpdateStatus(ctx, existingSSHKey, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("Updating sshkey status: %s", err)
	}

	return nil
}
