// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package sharing_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sg2v1alpha1 "github.com/vmware-tanzu/carvel-secretgen-controller/pkg/apis/secretgen2/v1alpha1"
	"github.com/vmware-tanzu/carvel-secretgen-controller/pkg/sharing"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeClient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func Test_Imports(t *testing.T) {
	t.Run("reconciling secret import-export pair result in creation of a copy of a secret into another namespace", func(t *testing.T) {
		sourceSecret := buildSourceSecret()
		secretImport := secretImportFor(sourceSecret)
		secretExport := secretExportFor(sourceSecret, secretImport.Namespace)
		secretExportReconciler, secretImportReconciler, k8sClient := importReconcilers(&sourceSecret, &secretExport, &secretImport)

		reconcileObject(t, secretExportReconciler, &secretExport)
		reconcileObject(t, secretImportReconciler, &secretImport)

		destinationSecret := corev1.Secret{}
		err := k8sClient.Get(context.Background(), namespacedNameFor(&secretImport), &destinationSecret)
		require.NoError(t, err)

		assert.Equal(t, sourceSecret.Type, destinationSecret.Type)
		assert.Equal(t, sourceSecret.Data, destinationSecret.Data)
		assert.Equal(t, secretImport.Namespace, destinationSecret.Namespace)
	})

	t.Run("reconciling secret import-export pair does not copy secret if names don't align", func(t *testing.T) {
		sourceSecret := buildSourceSecret()
		secretImport := secretImportFor(sourceSecret)
		secretImport.Name = "wrong name xx"

		secretExport := secretExportFor(sourceSecret, secretImport.Namespace)
		secretExportReconciler, secretImportReconciler, k8sClient := importReconcilers(&sourceSecret, &secretExport, &secretImport)

		reconcileObject(t, secretExportReconciler, &secretExport)
		reconcileObject(t, secretImportReconciler, &secretImport)

		destinationSecret := corev1.Secret{}
		err := k8sClient.Get(context.Background(), namespacedNameFor(&secretImport), &destinationSecret)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func secretImportFor(sourceSecret corev1.Secret) sg2v1alpha1.SecretImport {
	return sg2v1alpha1.SecretImport{
		ObjectMeta: metav1.ObjectMeta{
			Name:      sourceSecret.Name,
			Namespace: "test-target",
		},
		Spec: sg2v1alpha1.SecretImportSpec{
			FromNamespace: sourceSecret.Namespace,
		},
	}
}

func importReconcilers(objects ...runtime.Object) (secretExportReconciler *sharing.SecretExportReconciler, secretImportReconciler *sharing.SecretImportReconciler, k8sClient client.Client) {
	k8sClient = fakeClient.NewFakeClient(objects...)
	secretExports := sharing.NewSecretExportsWarmedUp(sharing.NewSecretExports(k8sClient, testLogr))
	secretExportReconciler = sharing.NewSecretExportReconciler(k8sClient, secretExports, testLogr)
	secretExports.WarmUpFunc = secretExportReconciler.WarmUp
	secretImportReconciler = sharing.NewSecretImportReconciler(k8sClient, secretExports, testLogr)
	return
}
