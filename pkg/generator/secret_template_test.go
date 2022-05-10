// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package generator_test

import (
	"context"
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sg2v1alpha1 "github.com/vmware-tanzu/carvel-secretgen-controller/pkg/apis/secretgen2/v1alpha1"
	"github.com/vmware-tanzu/carvel-secretgen-controller/pkg/client2/clientset/versioned/scheme"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeClient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/vmware-tanzu/carvel-secretgen-controller/pkg/generator"
)

func Test_SecretTemplate(t *testing.T) {
	t.Run("reconciling secret template", func(t *testing.T) {
		secret := Secret("existingSecret", map[string]string{
			"inputKey1": "value1",
			"inputKey2": "value2",
		})

		secretTemplate := secretTemplate("secretTemplate", secret, map[string]string{
			"key1": "{ .creds.data.inputKey1 }",
			"key2": "{ .creds.data.inputKey2 }",
		}, map[string]string{
			"key3": "value3",
		})

		secretTemplateReconciler, k8sClient := importReconcilers(&secret, &secretTemplate)

		reconcileObject(t, secretTemplateReconciler, &secretTemplate)

		createdSecret := corev1.Secret{}
		err := k8sClient.Get(context.Background(), namespacedNameFor(&secretTemplate), &createdSecret)
		require.NoError(t, err)

		key1, _ := base64.StdEncoding.DecodeString(string(createdSecret.Data["key1"]))
		key2, _ := base64.StdEncoding.DecodeString(string(createdSecret.Data["key2"]))
		key3, _ := createdSecret.StringData["key3"]

		assert.Equal(t, []byte("value1"), key1)
		assert.Equal(t, []byte("value2"), key2)
		assert.Equal(t, "value3", key3)
	})
}

func secretTemplate(name string, inputResource corev1.Secret, dataExpressions map[string]string, stringDataExpressions map[string]string) sg2v1alpha1.SecretTemplate {
	return sg2v1alpha1.SecretTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "test",
		},
		Spec: sg2v1alpha1.SecretTemplateSpec{
			InputResources: []sg2v1alpha1.InputResource{{
				Name: "creds",
				Ref: sg2v1alpha1.InputResourceRef{
					APIVersion: inputResource.APIVersion,
					Kind:       inputResource.Kind,
					Name:       inputResource.Name,
				},
			}},
			JSONPathTemplate: sg2v1alpha1.JSONPathTemplate{
				Data:       dataExpressions,
				StringData: stringDataExpressions,
			},
		},
	}
}

func Secret(name string, stringData map[string]string) corev1.Secret {
	data := map[string][]byte{}

	for key, val := range stringData {
		data[key] = []byte(val)
	}

	return corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "test",
		},
		Data: data,
	}
}

//TODO this is all copied and pasted from helpers in pkg/shared
func importReconcilers(objects ...client.Object) (secretTemplateReconciler *generator.SecretTemplateReconciler, k8sClient client.Client) {
	sg2v1alpha1.AddToScheme(scheme.Scheme)
	corev1.AddToScheme(scheme.Scheme)
	testLogr := zap.New(zap.UseDevMode(true))
	k8sClient = fakeClient.NewClientBuilder().WithObjects(objects...).WithScheme(scheme.Scheme).Build()
	secretTemplateReconciler = generator.NewSecretTemplateReconciler(k8sClient, testLogr)
	return
}

type reconcilerFunc interface {
	Reconcile(context.Context, reconcile.Request) (reconcile.Result, error)
}

func reconcileObject(t *testing.T, recon reconcilerFunc, object client.Object) {
	status, err := recon.Reconcile(context.Background(), reconcileRequestFor(object))
	require.NoError(t, err)
	require.False(t, status.Requeue)
}

func reconcileRequestFor(object client.Object) reconcile.Request {
	return reconcile.Request{NamespacedName: namespacedNameFor(object)}
}

func namespacedNameFor(object client.Object) types.NamespacedName {
	return types.NamespacedName{
		Name:      object.GetName(),
		Namespace: object.GetNamespace(),
	}
}
