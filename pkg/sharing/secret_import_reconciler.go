// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package sharing

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	sgv1alpha1 "github.com/vmware-tanzu/carvel-secretgen-controller/pkg/apis/secretgen/v1alpha1"
	sg2v1alpha1 "github.com/vmware-tanzu/carvel-secretgen-controller/pkg/apis/secretgen2/v1alpha1"
	"github.com/vmware-tanzu/carvel-secretgen-controller/pkg/reconciler"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// SecretImportReconciler creates an imported Secret if it was exported.
type SecretImportReconciler struct {
	client        client.Client
	secretExports SecretExportsProvider
	log           logr.Logger
}

var _ reconcile.Reconciler = &SecretImportReconciler{}

// NewSecretImportReconciler constructs SecretImportReconciler.
func NewSecretImportReconciler(client client.Client,
	secretExports SecretExportsProvider, log logr.Logger) *SecretImportReconciler {
	return &SecretImportReconciler{client, secretExports, log}
}

func (r *SecretImportReconciler) AttachWatches(controller controller.Controller) error {
	err := controller.Watch(&source.Kind{Type: &sg2v1alpha1.SecretImport{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return fmt.Errorf("Watching secret request: %s", err)
	}

	// Watch secrets and enqueue for same named SecretImport
	// to make sure imported secret is up-to-date
	err = controller.Watch(&source.Kind{Type: &corev1.Secret{}}, handler.EnqueueRequestsFromMapFunc(
		func(a client.Object) []reconcile.Request {
			return []reconcile.Request{
				{NamespacedName: types.NamespacedName{
					Name:      a.GetName(),
					Namespace: a.GetNamespace(),
				}},
			}
		},
	))
	if err != nil {
		return err
	}

	// Watch SecretExport and enqueue for related SecretImport
	// based on export namespace configuration
	err = controller.Watch(&source.Kind{Type: &sg2v1alpha1.SecretExport{}}, &enqueueSecretExportToSecret{
		SecretExports: r.secretExports,
		Log:           r.log,

		ToRequests: func(_ client.Object) []reconcile.Request {
			var secretReqList sg2v1alpha1.SecretImportList

			// TODO expensive call on every secret export update
			err := r.client.List(context.TODO(), &secretReqList)
			if err != nil {
				// TODO what should we really do here?
				r.log.Error(err, "Failed fetching list of all secret requests")
				return nil
			}

			var result []reconcile.Request
			for _, req := range secretReqList.Items {
				result = append(result, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      req.Name,
						Namespace: req.Namespace,
					},
				})
			}

			r.log.Info("Planning to reconcile matched secret requests",
				"all", len(secretReqList.Items))

			return result
		},
	})
	if err != nil {
		return err
	}

	// Watch namespaces partly so that we cache them because we might be doing a lot of lookups
	// note that for now we are using the same enqueueDueToNamespaceChange as the secretReconciler
	return controller.Watch(&source.Kind{Type: &corev1.Namespace{}}, &enqueueDueToNamespaceChange{
		ToRequests: r.mapNamespaceToSecretImports,
		Log:        r.log,
	})
}

func (r *SecretImportReconciler) mapNamespaceToSecretImports(ns client.Object) []reconcile.Request {
	var secretImportList sg2v1alpha1.SecretImportList
	err := r.client.List(context.Background(), &secretImportList, client.InNamespace(ns.GetName()))
	if err != nil {
		// TODO what should we really do here?
		r.log.Error(err, "Failed fetching list of all SecretImports")
		return nil
	}

	var result []reconcile.Request
	for _, secretImport := range secretImportList.Items {
		result = append(result, reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      secretImport.Name,
				Namespace: secretImport.Namespace,
			},
		})
	}

	r.log.Info("Planning to reconcile matched SecretImports",
		"count", len(secretImportList.Items))

	return result
}

// Reconcile is the entrypoint for incoming requests from k8s
func (r *SecretImportReconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.WithValues("request", request)

	var secretImport sg2v1alpha1.SecretImport

	err := r.client.Get(ctx, request.NamespacedName, &secretImport)
	if err != nil {
		if errors.IsNotFound(err) {
			// Do not requeue as there is nothing to do when request is deleted
			return reconcile.Result{}, nil
		}
		// Requeue to try to fetch request again
		return reconcile.Result{Requeue: true}, err
	}

	if secretImport.DeletionTimestamp != nil {
		// Do not requeue as there is nothing to do
		// Associated secret has owned ref so it's going to be deleted
		return reconcile.Result{}, nil
	}

	status := &reconciler.Status{
		secretImport.Status.GenericStatus,
		func(st sgv1alpha1.GenericStatus) { secretImport.Status.GenericStatus = st },
	}

	status.SetReconciling(secretImport.ObjectMeta)

	reconcileResult, reconcileErr := status.WithReconcileCompleted(r.reconcile(ctx, secretImport, log))

	err = r.updateStatus(ctx, secretImport)
	if err != nil {
		return reconcile.Result{}, err
	}

	return reconcileResult, reconcileErr
}

func (r *SecretImportReconciler) reconcile(
	ctx context.Context, secretImport sg2v1alpha1.SecretImport,
	log logr.Logger) (reconcile.Result, error) {

	err := secretImport.Validate()
	if err != nil {
		// Do not requeue as there is nothing this controller can do until secret request is fixed
		return reconcile.Result{}, reconciler.TerminalReconcileErr{err}
	}

	log.Info("Reconciling")

	nsName := secretImport.Namespace
	query := types.NamespacedName{
		Name: nsName,
	}
	namespace := corev1.Namespace{}
	err = r.client.Get(ctx, query, &namespace)
	var fromNamespaceAnnotations map[string]string
	if err == nil {
		fromNamespaceAnnotations = namespace.GetAnnotations()
	}

	matcher := SecretMatcher{
		FromName:                 secretImport.Name,
		FromNamespace:            secretImport.Spec.FromNamespace,
		FromNamespaceAnnotations: fromNamespaceAnnotations,
		ToNamespace:              secretImport.Namespace,
	}

	nscheck := makeNamespaceWildcardExclusionCheck(ctx, r.client, log)
	secrets := r.secretExports.MatchedSecretsForImport(matcher, nscheck)

	switch len(secrets) {
	case 0:
		err := r.deleteAssociatedSecret(ctx, secretImport)
		if err != nil {
			// Requeue to try to delete a bit later
			return reconcile.Result{Requeue: true}, err
		}
		// Do not requeue since export is not offered
		return reconcile.Result{}, reconciler.TerminalReconcileErr{fmt.Errorf("No matching export/secret")}

	case 1:
		return r.copyAssociatedSecret(ctx, secretImport, secrets[0])

	default:
		panic("Internal inconsistency: multiple exports/secrets matched one ns+name")
	}
}

func (r *SecretImportReconciler) copyAssociatedSecret(
	ctx context.Context, imp sg2v1alpha1.SecretImport, srcSecret *corev1.Secret) (reconcile.Result, error) {

	secret := reconciler.NewSecret(&imp, nil)
	secret.ApplySecret(*srcSecret)

	err := r.client.Create(ctx, secret.AsSecret())
	switch {
	case err == nil:
		// Do not requeue since we copied secret successfully
		return reconcile.Result{}, nil

	case errors.IsAlreadyExists(err):
		var existingSecret corev1.Secret
		existingSecretNN := types.NamespacedName{Namespace: imp.Namespace, Name: imp.Name}

		err := r.client.Get(ctx, existingSecretNN, &existingSecret)
		if err != nil {
			// Requeue to try to fetch a bit later
			return reconcile.Result{Requeue: true}, fmt.Errorf("Getting imported secret: %s", err)
		}

		secret.AssociateExistingSecret(existingSecret)

		err = r.client.Update(ctx, secret.AsSecret())
		if err != nil {
			// Requeue to try to update a bit later
			return reconcile.Result{Requeue: true}, fmt.Errorf("Updating imported secret: %s", err)
		}

		// Do not requeue since we copied secret successfully
		return reconcile.Result{}, nil

	default:
		// Requeue to try to create a bit later
		return reconcile.Result{Requeue: true}, fmt.Errorf("Creating imported secret: %s", err)
	}
}

func (r *SecretImportReconciler) deleteAssociatedSecret(
	ctx context.Context, imp sg2v1alpha1.SecretImport) error {

	var secret corev1.Secret
	secretNN := types.NamespacedName{Namespace: imp.Namespace, Name: imp.Name}

	err := r.client.Get(ctx, secretNN, &secret)
	if err != nil {
		return nil
	}

	err = r.client.Delete(ctx, &secret)
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("Deleting associated secret: %s", err)
	}
	return nil
}

func (r *SecretImportReconciler) updateStatus(
	ctx context.Context, imp sg2v1alpha1.SecretImport) error {

	err := r.client.Status().Update(ctx, &imp)
	if err != nil {
		return fmt.Errorf("Updating secret request status: %s", err)
	}
	return nil
}
