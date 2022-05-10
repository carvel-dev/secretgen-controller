// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package generator

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sg2v1alpha1 "github.com/vmware-tanzu/carvel-secretgen-controller/pkg/apis/secretgen2/v1alpha1"
	"github.com/vmware-tanzu/carvel-secretgen-controller/pkg/client2/clientset/versioned/scheme"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeClient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var testLogr logr.Logger

func init() {
	sg2v1alpha1.AddToScheme(scheme.Scheme)
	testLogr = zap.New(zap.UseDevMode(true))
}

func Test_SecretTemplate(t *testing.T) {
	t.Run("reconciling secret import-export pair result in creation of a copy of a secret into another namespace", func(t *testing.T) {
		secret := Secret("secret", map[string]string{
			"inputKey1": "value1",
			"inputKey2": "value2",
		})

		secretTemplate := secretTemplate("template", secret, map[string]string{
			"key1": "{.creds.data.inputKey1}",
			"key2": "{.creds.data.inputKey2}",
			"key3": "value3",
		})

		secretTemplateReconciler, k8sClient := importReconcilers(&secret, &secretTemplate)

		reconcileObject(t, secretExportReconciler, &secretExport)
		reconcileObject(t, secretTemplateReconciler, &secretTemplate)

		destinationSecret := corev1.Secret{}
		err := k8sClient.Get(context.Background(), namespacedNameFor(&secretTemplate), &destinationSecret)
		require.NoError(t, err)

		assert.Equal(t, sourceSecret.Type, destinationSecret.Type)
		assert.Equal(t, sourceSecret.Data, destinationSecret.Data)
		assert.Equal(t, secretTemplate.Namespace, destinationSecret.Namespace)
	})
}

func secretTemplate(name string, inputResource corev1.Secret, springDataExpressions map[string]string) sg2v1alpha1.SecretTemplate {
	return sg2v1alpha1.SecretTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "test",
		},
		Spec: sg2v1alpha1.SecretTemplateSpec{
			InputResources: []sg2v1alpha1.InputResource{{
				Name: "creds",
				Ref: sg2v1alpha1.InputResourceRef{
					ApiVersion: inputResource.APIVersion,
					Kind:       inputResource.Kind,
					Name:       inputResource.Name,
				},
			}},
			JsonPathTemplate: sg2v1alpha1.JsonPathTemplate{
				StringData: springDataExpressions,
			},
		},
	}
}

func Secret(name string, stringData map[string]string) corev1.Secret {
	return corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "test-target",
		},
		StringData: stringData,
	}
}

func importReconcilers(objects ...runtime.Object) (secretTemplateReconciler *SecretTemplateReconciler, k8sClient client.Client) {
	k8sClient = fakeClient.NewFakeClient(objects...)
	secretTemplateReconciler = NewSecretTemplateReconciler(k8sClient, testLogr)
	return
}
