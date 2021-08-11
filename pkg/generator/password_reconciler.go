// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package generator

import (
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
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type PasswordReconciler struct {
	sgClient   sgclient.Interface
	coreClient kubernetes.Interface
	log        logr.Logger
}

var _ reconcile.Reconciler = &PasswordReconciler{}

func NewPasswordReconciler(sgClient sgclient.Interface,
	coreClient kubernetes.Interface, log logr.Logger) *PasswordReconciler {
	return &PasswordReconciler{sgClient, coreClient, log}
}

func (r *PasswordReconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	log := r.log.WithValues("request", request)

	password, err := r.sgClient.SecretgenV1alpha1().Passwords(request.Namespace).Get(request.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("Not found")
			return reconcile.Result{}, nil
		}
		return reconcile.Result{Requeue: true}, err
	}

	if password.DeletionTimestamp != nil {
		// Nothing to do
		return reconcile.Result{}, nil
	}

	status := &reconciler.Status{
		password.Status.GenericStatus,
		func(st sgv1alpha1.GenericStatus) { password.Status.GenericStatus = st },
	}

	status.SetReconciling(password.ObjectMeta)
	defer r.updateStatus(password)

	return status.WithReconcileCompleted(r.reconcile(password))
}

func (r *PasswordReconciler) reconcile(password *sgv1alpha1.Password) (reconcile.Result, error) {
	_, err := r.coreClient.CoreV1().Secrets(password.Namespace).Get(password.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return r.createSecret(password)
		}
		return reconcile.Result{Requeue: true}, err
	}

	return reconcile.Result{}, nil
}

func (r *PasswordReconciler) createSecret(password *sgv1alpha1.Password) (reconcile.Result, error) {
	passwordStr, err := r.generate(password)
	if err != nil {
		return reconcile.Result{Requeue: true}, err
	}

	values := map[string][]byte{
		sgv1alpha1.PasswordSecretKey: []byte(passwordStr),
	}

	secret := reconciler.NewSecret(password, values)

	defaultTemplate := sgv1alpha1.SecretTemplate{
		Type: sgv1alpha1.PasswordSecretDefaultType,
		StringData: map[string]string{
			sgv1alpha1.PasswordSecretDefaultKey: expansion.Variable(sgv1alpha1.PasswordSecretKey),
		},
	}

	err = secret.ApplyTemplates(defaultTemplate, password.Spec.SecretTemplate)
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

type passwordParams struct {
	Length int `yaml:"length"`
}

func (r *PasswordReconciler) generate(password *sgv1alpha1.Password) (string, error) {
	gen := cfgtypes.NewPasswordGenerator()

	genParams := passwordParams{Length: password.Spec.Length}
	if genParams.Length == 0 {
		genParams.Length = 40
	}

	passwordVal, err := gen.Generate(genParams)
	if err != nil {
		return "", err
	}

	return passwordVal.(string), nil
}

func (r *PasswordReconciler) updateStatus(password *sgv1alpha1.Password) error {
	existingPassword, err := r.sgClient.SecretgenV1alpha1().Passwords(password.Namespace).Get(password.Name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("Fetching password: %s", err)
	}

	existingPassword.Status = password.Status

	_, err = r.sgClient.SecretgenV1alpha1().Passwords(existingPassword.Namespace).UpdateStatus(existingPassword)
	if err != nil {
		return fmt.Errorf("Updating password status: %s", err)
	}

	return nil
}
