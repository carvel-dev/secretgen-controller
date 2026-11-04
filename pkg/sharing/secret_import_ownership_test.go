// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package sharing_test

import (
	"context"
	"testing"

	sgv1alpha1 "carvel.dev/secretgen-controller/pkg/apis/secretgen/v1alpha1"
	sg2v1alpha1 "carvel.dev/secretgen-controller/pkg/apis/secretgen2/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func Test_ImportOwnership(t *testing.T) {
	unownedSecret := func(imp sg2v1alpha1.SecretImport) corev1.Secret {
		return corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: imp.Name, Namespace: imp.Namespace},
			Data:       map[string][]byte{"key": []byte("operator-managed")},
		}
	}

	withUID := func(imp sg2v1alpha1.SecretImport) sg2v1alpha1.SecretImport {
		imp.UID = types.UID("import-uid")
		return imp
	}

	getImport := func(t *testing.T, c client.Client, imp sg2v1alpha1.SecretImport) sg2v1alpha1.SecretImport {
		var got sg2v1alpha1.SecretImport
		require.NoError(t, c.Get(context.Background(), namespacedNameFor(&imp), &got))
		return got
	}

	requireReconcileFailed := func(t *testing.T, imp sg2v1alpha1.SecretImport, msg string) {
		require.Len(t, imp.Status.Conditions, 1)
		assert.Equal(t, sgv1alpha1.ReconcileFailed, imp.Status.Conditions[0].Type)
		assert.Contains(t, imp.Status.Conditions[0].Message, msg)
	}

	t.Run("does not overwrite an existing secret not controlled by the import", func(t *testing.T) {
		sourceSecret := buildSourceSecret()
		secretImport := withUID(secretImportFor(sourceSecret))
		secretExport := secretExportFor(sourceSecret, secretImport.Namespace)
		existing := unownedSecret(secretImport)
		exportRecon, importRecon, k8sClient := importReconcilers(&sourceSecret, &secretExport, &secretImport, &existing)

		reconcileObject(t, exportRecon, &secretExport)
		reconcileObject(t, importRecon, &secretImport)

		var got corev1.Secret
		require.NoError(t, k8sClient.Get(context.Background(), namespacedNameFor(&existing), &got))
		assert.Equal(t, existing.Data, got.Data)
		assert.Empty(t, got.OwnerReferences)

		requireReconcileFailed(t, getImport(t, k8sClient, secretImport), "not controlled by this SecretImport")
	})

	t.Run("does not overwrite an existing secret controlled by a different owner", func(t *testing.T) {
		sourceSecret := buildSourceSecret()
		secretImport := withUID(secretImportFor(sourceSecret))
		secretExport := secretExportFor(sourceSecret, secretImport.Namespace)
		existing := unownedSecret(secretImport)
		isController := true
		existing.OwnerReferences = []metav1.OwnerReference{{
			APIVersion: sg2v1alpha1.SchemeGroupVersion.String(),
			Kind:       "SecretImport",
			Name:       secretImport.Name,
			UID:        types.UID("some-other-uid"),
			Controller: &isController,
		}}
		exportRecon, importRecon, k8sClient := importReconcilers(&sourceSecret, &secretExport, &secretImport, &existing)

		reconcileObject(t, exportRecon, &secretExport)
		reconcileObject(t, importRecon, &secretImport)

		var got corev1.Secret
		require.NoError(t, k8sClient.Get(context.Background(), namespacedNameFor(&existing), &got))
		assert.Equal(t, existing.Data, got.Data)
	})

	t.Run("does not delete an existing secret not controlled by the import when no export matches", func(t *testing.T) {
		sourceSecret := buildSourceSecret()
		secretImport := withUID(secretImportFor(sourceSecret))
		secretImport.Spec.FromNamespace = "nonexistent-namespace"
		existing := unownedSecret(secretImport)
		_, importRecon, k8sClient := importReconcilers(&secretImport, &existing)

		reconcileObject(t, importRecon, &secretImport)

		var got corev1.Secret
		require.NoError(t, k8sClient.Get(context.Background(), namespacedNameFor(&existing), &got))
		assert.Equal(t, existing.Data, got.Data)

		requireReconcileFailed(t, getImport(t, k8sClient, secretImport), "No matching export/secret")
	})

	t.Run("still updates a secret controlled by the import when the source changes", func(t *testing.T) {
		sourceSecret := buildSourceSecret()
		secretImport := withUID(secretImportFor(sourceSecret))
		secretExport := secretExportFor(sourceSecret, secretImport.Namespace)
		exportRecon, importRecon, k8sClient := importReconcilers(&sourceSecret, &secretExport, &secretImport)

		reconcileObject(t, exportRecon, &secretExport)
		reconcileObject(t, importRecon, &secretImport)

		updatedData := map[string][]byte{corev1.DockerConfigJsonKey: []byte(`{"auths":{"server":{"auth":"newAuth"}}}`)}
		var src corev1.Secret
		require.NoError(t, k8sClient.Get(context.Background(), namespacedNameFor(&sourceSecret), &src))
		src.Data = updatedData
		require.NoError(t, k8sClient.Update(context.Background(), &src))

		reconcileObject(t, exportRecon, &secretExport)
		reconcileObject(t, importRecon, &secretImport)

		var got corev1.Secret
		require.NoError(t, k8sClient.Get(context.Background(), namespacedNameFor(&secretImport), &got))
		assert.Equal(t, updatedData, got.Data)
	})

	t.Run("still deletes a secret controlled by the import when the export is removed", func(t *testing.T) {
		sourceSecret := buildSourceSecret()
		secretImport := withUID(secretImportFor(sourceSecret))
		secretExport := secretExportFor(sourceSecret, secretImport.Namespace)
		exportRecon, importRecon, k8sClient := importReconcilers(&sourceSecret, &secretExport, &secretImport)

		reconcileObject(t, exportRecon, &secretExport)
		reconcileObject(t, importRecon, &secretImport)

		var got corev1.Secret
		require.NoError(t, k8sClient.Get(context.Background(), namespacedNameFor(&secretImport), &got))

		require.NoError(t, k8sClient.Delete(context.Background(), &secretExport))
		reconcileObject(t, exportRecon, &secretExport)
		reconcileObject(t, importRecon, &secretImport)

		err := k8sClient.Get(context.Background(), namespacedNameFor(&secretImport), &got)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}
