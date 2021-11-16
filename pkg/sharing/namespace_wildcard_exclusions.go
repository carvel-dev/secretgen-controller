// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package sharing

import (
	"context"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// makeNamespaceWildcardExclusionCheck returns a function that uses reconciler-level tools (k8s client, logger, context) to
// check the presence of a namespace annotation that we mostly only care about in the inner workings of SecretExport.
func makeNamespaceWildcardExclusionCheck(ctx context.Context,
	kubernetesClient client.Client,
	log logr.Logger) NamespaceWildcardExclusionCheck {
	return func(nsName string) bool {
		query := types.NamespacedName{
			Name: nsName,
		}
		namespace := corev1.Namespace{}
		err := kubernetesClient.Get(ctx, query, &namespace)
		if err != nil {
			log.Error(err, "Called to check annotation on a namespace but couldn't find:", "namespace", nsName)
			return false
		}
		return nsHasExclusionAnnotation(namespace)
	}
}

func nsHasExclusionAnnotation(ns corev1.Namespace) bool {
	_, excluded := ns.Annotations["secretgen.carvel.dev/excluded-from-wildcard-matching"]
	return excluded
}

// enqueueNamespaceToSecret is a custom handler that is optimized for tracking
// Namespace annotation change events. It tries to result in minimum number of
// Secret reconcile requests. Used in both SecretImport and Secret Reconcilers.
type enqueueNamespaceToSecret struct {
	ToRequests handler.MapFunc
	Log        logr.Logger
}

// Create doesn't do anything
func (e *enqueueNamespaceToSecret) Create(evt event.CreateEvent, q workqueue.RateLimitingInterface) {}

// Update checks whether the exclusion annotation has been added or removed and then queues the secrets in that namespace
func (e *enqueueNamespaceToSecret) Update(evt event.UpdateEvent, q workqueue.RateLimitingInterface) {
	typedNsOld, okOld := evt.ObjectOld.(*corev1.Namespace)
	typedNsNew, okNew := evt.ObjectNew.(*corev1.Namespace)
	if okOld && okNew && (nsHasExclusionAnnotation(*typedNsOld) == nsHasExclusionAnnotation(*typedNsNew)) {
		return // Skip when exclusion annotation did not change
	}

	e.mapAndEnqueue(q, evt.ObjectNew)
}

// Delete doesn't do anything
func (e *enqueueNamespaceToSecret) Delete(evt event.DeleteEvent, q workqueue.RateLimitingInterface) {}

// Generic doesn't do anything
func (e *enqueueNamespaceToSecret) Generic(evt event.GenericEvent, q workqueue.RateLimitingInterface) {
}

func (e *enqueueNamespaceToSecret) mapAndEnqueue(q workqueue.RateLimitingInterface, object client.Object) {
	for _, req := range e.ToRequests(object) {
		q.Add(req)
	}
}

// mapNamespaceToSecret implements the logic inside of the enqueueNamespaceToSecret.mapAndEnqueue function.
// the reconcilers that us the other objects in this file have instance methods that delegate all the logic to this method
func mapNamespaceToSecret(ns client.Object, kubernetesClient client.Client, log logr.Logger) []reconcile.Request {
	var secretList corev1.SecretList
	err := kubernetesClient.List(context.Background(), &secretList, client.InNamespace(ns.GetName()))
	if err != nil {
		// TODO what should we really do here?
		log.Error(err, "Failed fetching list of all secrets")
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
	log.Info("Planning to reconcile matched secrets",
		"count", len(secretList.Items))

	return result
}
