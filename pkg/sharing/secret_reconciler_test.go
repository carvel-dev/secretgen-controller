// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package sharing_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sg2v1alpha1 "github.com/vmware-tanzu/carvel-secretgen-controller/pkg/apis/secretgen2/v1alpha1"
	"github.com/vmware-tanzu/carvel-secretgen-controller/pkg/sharing"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeClient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func Test_SecretReconciler_updatesStatus(t *testing.T) {
	sg2v1alpha1.AddToScheme(scheme.Scheme)
	testLogr := zap.New(zap.UseDevMode(true))

	resourcesUnderTest := func() (sourceSecret corev1.Secret, placeholderSecret corev1.Secret, secretExport sg2v1alpha1.SecretExport) {
		sourceSecret = corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Secret",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-secret",
				Namespace: "test-source",
			},
			Data: map[string][]byte{
				corev1.DockerConfigJsonKey: []byte(`{"auths":{"server":{"username":"correctUser","password":"correctPassword","auth":"correctAuth"}}}`),
			},
			Type: "kubernetes.io/dockerconfigjson",
		}
		placeholderSecret = corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Secret",
				APIVersion: "v1",
			},
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
		secretExport = sg2v1alpha1.SecretExport{
			TypeMeta: metav1.TypeMeta{
				Kind:       "SecretExport",
				APIVersion: "secretgen.k14s.io/v1alpha1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-secret",
				Namespace: "test-source",
			},
			Spec: sg2v1alpha1.SecretExportSpec{
				ToNamespaces: []string{"test-target"},
			},
		}
		return
	}

	reconcilersUnderTest := func(objects ...runtime.Object) (secretExportReconciler *sharing.SecretExportReconciler, secretReconciler *sharing.SecretReconciler, k8sClient client.Client) {
		secretExports := sharing.NewSecretExportsWarmedUp(sharing.NewSecretExports(testLogr))
		k8sClient = fakeClient.NewFakeClient(objects...)
		secretExportReconciler = sharing.NewSecretExportReconciler(k8sClient, secretExports, testLogr)
		secretReconciler = sharing.NewSecretReconciler(k8sClient, secretExports, testLogr)
		secretExports.WarmUpFunc = secretExportReconciler.WarmUp

		return
	}

	t.Run("happy path", func(t *testing.T) {
		sourceSecret, placeholderSecret, secretExport := resourcesUnderTest()
		secretExportReconciler, secretReconciler, k8sClient := reconcilersUnderTest(&sourceSecret, &placeholderSecret, &secretExport)

		reconcileObject(t, secretExportReconciler, &secretExport)
		reconcileObject(t, secretReconciler, &placeholderSecret)

		reload(t, &placeholderSecret, k8sClient)
		assert.Equal(t, sourceSecret.Data[".dockerconfigjson"], placeholderSecret.Data[".dockerconfigjson"])
		assert.NotNil(t, placeholderSecret.ObjectMeta.Annotations["secretgen.carvel.dev/status"])
		var observedStatus map[string]interface{}
		require.NoError(t, json.Unmarshal([]byte(placeholderSecret.ObjectMeta.Annotations["secretgen.carvel.dev/status"]), &observedStatus))
		expectedStatus := map[string]interface{}{"conditions": []interface{}{map[string]interface{}{"status": "True", "type": "ReconcileSucceeded"}}, "secretNames": []interface{}{"test-source/test-secret"}}
		assert.Equal(t, expectedStatus, observedStatus)

		assert.Equal(t, 0, len(secretExport.Status.Conditions))
		reload(t, &secretExport, k8sClient)
		assert.Equal(t, 1, len(secretExport.Status.Conditions))
		assert.Equal(t, secretExport.Status.FriendlyDescription, "Reconcile succeeded")
	})

	t.Run("wrong placeholder secret type gets informative status", func(t *testing.T) {
		sourceSecret, placeholderSecret, secretExport := resourcesUnderTest()
		placeholderSecret.Type = ""
		secretExportReconciler, secretReconciler, k8sClient := reconcilersUnderTest(&sourceSecret, &placeholderSecret, &secretExport)

		reconcileObject(t, secretExportReconciler, &secretExport)
		reconcileObject(t, secretReconciler, &placeholderSecret)

		// placeholder secret should have its original contents for auths and a helpful status message
		originalPlaceholderData := make([]byte, len(placeholderSecret.Data[".dockerconfigjson"]))
		copy(originalPlaceholderData, placeholderSecret.Data[".dockerconfigjson"])

		reload(t, &placeholderSecret, k8sClient)
		assert.Equal(t, originalPlaceholderData, placeholderSecret.Data[".dockerconfigjson"])
		assert.NotNil(t, placeholderSecret.ObjectMeta.Annotations["secretgen.carvel.dev/status"])
		var observedStatus map[string]interface{}
		require.NoError(t, json.Unmarshal([]byte(placeholderSecret.ObjectMeta.Annotations["secretgen.carvel.dev/status"]), &observedStatus))
		expectedStatus := map[string]interface{}{"conditions": []interface{}{map[string]interface{}{"message": "Expected secret to have type=corev1.SecretTypeDockerConfigJson, but did not", "status": "True", "type": "ReconcileFailed"}}}
		assert.Equal(t, expectedStatus, observedStatus)

		// from secret export's perspective it still reconciled successfully.
		reload(t, &secretExport, k8sClient)
		assert.Equal(t, secretExport.Status.FriendlyDescription, "Reconcile succeeded")
	})

	t.Run("wrong source secret type gets informative status", func(t *testing.T) {
		sourceSecret, placeholderSecret, secretExport := resourcesUnderTest()
		sourceSecret.Type = ""
		secretExportReconciler, secretReconciler, k8sClient := reconcilersUnderTest(&sourceSecret, &placeholderSecret, &secretExport)

		reconcileObject(t, secretExportReconciler, &secretExport)
		reconcileObject(t, secretReconciler, &placeholderSecret)

		// placeholder secret should have its original contents for auths
		originalPlaceholderData := make([]byte, len(placeholderSecret.Data[".dockerconfigjson"]))
		copy(originalPlaceholderData, placeholderSecret.Data[".dockerconfigjson"])

		reload(t, &placeholderSecret, k8sClient)
		assert.Equal(t, originalPlaceholderData, placeholderSecret.Data[".dockerconfigjson"])
		assert.NotNil(t, placeholderSecret.ObjectMeta.Annotations["secretgen.carvel.dev/status"])
		var observedStatus map[string]interface{}
		require.NoError(t, json.Unmarshal([]byte(placeholderSecret.ObjectMeta.Annotations["secretgen.carvel.dev/status"]), &observedStatus))
		// Note placeholder secret Status has no "secretNames" key
		expectedStatus := map[string]interface{}{"conditions": []interface{}{map[string]interface{}{"status": "True", "type": "ReconcileSucceeded"}}}
		assert.Equal(t, expectedStatus, observedStatus)

		// from secret export's perspective it still reconciled successfully.
		reload(t, &secretExport, k8sClient)
		assert.Equal(t, secretExport.Status.FriendlyDescription, "Reconcile succeeded")
	})
}

// reload asks the Kubernetes runtime client to re-populate our object
// otherwise our local copy won't reflect changes made during controller reconcile calls, etc.
func reload(t *testing.T, object client.Object, k8sClient client.Client) {
	err := k8sClient.Get(context.Background(), namespacedNameFor(object), object)
	require.NoError(t, err)
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
