// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package sharing_test

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"
	sg2v1alpha1 "github.com/vmware-tanzu/carvel-secretgen-controller/pkg/apis/secretgen2/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var testLogr logr.Logger

func init() {
	sg2v1alpha1.AddToScheme(scheme.Scheme)
	testLogr = zap.New(zap.UseDevMode(true))
}

func buildSourceSecret() corev1.Secret {
	return corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-secret",
			Namespace: "test-source",
		},
		Data: map[string][]byte{
			corev1.DockerConfigJsonKey: []byte(`{"auths":{"server":{"username":"correctUser","password":"correctPassword","auth":"correctAuth"}}}`),
		},
		Type: "kubernetes.io/dockerconfigjson",
	}
}

func sourceAndPlaceholder() (sourceSecret corev1.Secret, placeholderSecret corev1.Secret) {
	sourceSecret = buildSourceSecret()
	placeholderSecret = corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "placeholder-secret",
			Namespace:   "test-target",
			Annotations: map[string]string{"secretgen.carvel.dev/image-pull-secret": ""},
		},
		Data: map[string][]byte{
			corev1.DockerConfigJsonKey: []byte(`{"auths":{}}`),
		},
		Type: "kubernetes.io/dockerconfigjson",
	}
	return
}

func secretExportFor(sourceSecret corev1.Secret, toNamespace string) sg2v1alpha1.SecretExport {
	return sg2v1alpha1.SecretExport{
		ObjectMeta: metav1.ObjectMeta{
			Name:      sourceSecret.Name,
			Namespace: sourceSecret.Namespace,
		},
		Spec: sg2v1alpha1.SecretExportSpec{
			ToNamespaces: []string{toNamespace},
		},
	}
}

func namespacedNameFor(object client.Object) types.NamespacedName {
	return types.NamespacedName{
		Name:      object.GetName(),
		Namespace: object.GetNamespace(),
	}
}

type reconciler interface {
	Reconcile(context.Context, reconcile.Request) (reconcile.Result, error)
}

func reconcileObject(t *testing.T, recon reconciler, object client.Object) {
	status, err := recon.Reconcile(context.Background(), reconcileRequestFor(object))
	require.NoError(t, err)
	require.False(t, status.Requeue)
}

func reconcileRequestFor(object client.Object) reconcile.Request {
	return reconcile.Request{NamespacedName: namespacedNameFor(object)}
}
