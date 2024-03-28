// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package generator

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"

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

// AttachWatches adds starts watches this reconciler requires.
func (r *PasswordReconciler) AttachWatches(controller controller.Controller) error {
	return controller.Watch(&source.Kind{Type: &sgv1alpha1.Password{}}, &handler.EnqueueRequestForObject{})
}

// Reconcile is the entrypoint for incoming requests from k8s
func (r *PasswordReconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.WithValues("request", request)

	password, err := r.sgClient.SecretgenV1alpha1().Passwords(request.Namespace).Get(ctx, request.Name, metav1.GetOptions{})
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
	defer r.updateStatus(ctx, password)

	return status.WithReconcileCompleted(r.reconcile(ctx, password))
}

func (r *PasswordReconciler) reconcile(ctx context.Context, password *sgv1alpha1.Password) (reconcile.Result, error) {
	_, err := r.coreClient.CoreV1().Secrets(password.Namespace).Get(ctx, password.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return r.createSecret(ctx, password)
		}
		return reconcile.Result{Requeue: true}, err
	}

	return reconcile.Result{}, nil
}

func (r *PasswordReconciler) createSecret(ctx context.Context, password *sgv1alpha1.Password) (reconcile.Result, error) {
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

	_, err = r.coreClient.CoreV1().Secrets(newSecret.Namespace).Create(ctx, newSecret, metav1.CreateOptions{})
	if err != nil {
		return reconcile.Result{Requeue: true}, err
	}

	return reconcile.Result{}, nil
}

func (r *PasswordReconciler) generate(password *sgv1alpha1.Password) (string, error) {
	const (
		lowerCharSet = "abcdefghijklmnopqrstuvwxyz"
		upperCharSet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
		digitSet     = "0123456789"
	)
	spec := password.Spec
	var generatedPassword strings.Builder

	//Set symbol character
	for i := 0; i < spec.Symbols; i++ {
		value, err := randomElement(spec.SymbolCharSet)
		if err != nil {
			return "", err
		}
		generatedPassword.WriteString(value)
	}

	//Set digit
	for i := 0; i < spec.Digits; i++ {
		value, err := randomElement(digitSet)
		if err != nil {
			return "", err
		}
		generatedPassword.WriteString(value)
	}

	//Set uppercase
	for i := 0; i < spec.UppercaseLetters; i++ {
		value, err := randomElement(upperCharSet)
		if err != nil {
			return "", err
		}
		generatedPassword.WriteString(value)
	}

	//Set lowercase
	for i := 0; i < spec.LowercaseLetters; i++ {
		value, err := randomElement(lowerCharSet)
		if err != nil {
			return "", err
		}
		generatedPassword.WriteString(value)
	}

	var allCharSet = lowerCharSet + upperCharSet + digitSet

	remainingLength := spec.Length - spec.Symbols - spec.Digits - spec.UppercaseLetters - spec.LowercaseLetters
	for i := 0; i < remainingLength; i++ {
		value, err := randomElement(allCharSet)
		if err != nil {
			return "", err
		}
		generatedPassword.WriteString(value)
	}

	inRune := []rune(generatedPassword.String())
	localCryptoShuffle(len(inRune), func(i, j int) {
		inRune[i], inRune[j] = inRune[j], inRune[i]
	})

	return string(inRune), nil
}

func randomElement(s string) (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(int64(len(s))))
	if err != nil {
		return "", err
	}
	return string(s[n.Int64()]), nil
}

func localCryptoShuffle(n int, swap func(i, j int)) {
	if n < 0 {
		panic("invalid argument to Shuffle")
	}

	// Fisher-Yates shuffle: https://en.wikipedia.org/wiki/Fisher%E2%80%93Yates_shuffle
	// Shuffle really ought not be called with n that doesn't fit in 32 bits.
	// Not only will it take a very long time, but with 2³¹! possible permutations,
	// there's no way that any PRNG can have a big enough internal state to
	// generate even a minuscule percentage of the possible permutations.
	// Nevertheless, the right API signature accepts an int n, so handle it as best we can.
	i := n - 1
	for ; i > 1<<31-1-1; i-- {
		j, _ := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
		swap(i, int(j.Int64()))
	}
	for ; i > 0; i-- {
		j, _ := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
		swap(i, int(j.Int64()))
	}
}

func (r *PasswordReconciler) updateStatus(ctx context.Context, password *sgv1alpha1.Password) error {
	existingPassword, err := r.sgClient.SecretgenV1alpha1().Passwords(password.Namespace).Get(ctx, password.Name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("fetching password: %s", err)
	}

	existingPassword.Status = password.Status

	_, err = r.sgClient.SecretgenV1alpha1().Passwords(existingPassword.Namespace).UpdateStatus(ctx, existingPassword, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("updating password status: %s", err)
	}

	return nil
}
