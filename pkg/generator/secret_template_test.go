// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package generator_test

import (
	"context"
	"reflect"
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

//TODO maybe this should be table tested.
func Test_SecretTemplate(t *testing.T) {
	t.Run("reconciling secret template with input from another secret", func(t *testing.T) {
		secret := Secret("existingSecret", map[string]string{
			"inputKey1": "value1",
			"inputKey2": "value2",
		})

		secretTemplate := secretTemplate("secretTemplate", "", map[string]client.Object{"creds": &secret}, map[string]string{
			"key1": "$( .creds.data.inputKey1 )",
			"key2": "$( .creds.data.inputKey2 )",
		}, map[string]string{
			"key3": "value3",
		})

		secretTemplateReconciler, k8sClient := importReconcilers(&secret, &secretTemplate)

		reconcileObject(t, secretTemplateReconciler, &secretTemplate)

		createdSecret := corev1.Secret{}
		err := k8sClient.Get(context.Background(), namespacedNameFor(&secretTemplate), &createdSecret)
		require.NoError(t, err)

		assert.Equal(t, []byte("value1"), createdSecret.Data["key1"])
		assert.Equal(t, []byte("value2"), createdSecret.Data["key2"])
		assert.Equal(t, "value3", createdSecret.StringData["key3"])
	})
}

func Test_SecretTemplate_Dynamic_InputResources(t *testing.T) {
	t.Run("reconciling secret template with input from another secret", func(t *testing.T) {
		first := ConfigMap("first", map[string]string{
			"next": "dynamic-secret-name",
		})

		second := Secret("dynamic-secret-name", map[string]string{
			"inputKey1": "value1",
		})

		secondInput := second.DeepCopy()
		secondInput.Name = "$(.first.data.next)"

		secretTemplate := secretTemplate(
			"secretTemplate",
			"",
			map[string]client.Object{
				"first":  &first,
				"second": &second,
			}, map[string]string{
				"key1": "$(.second.data.inputKey1)",
			}, map[string]string{})

		secretTemplateReconciler, k8sClient := importReconcilers(&first, &second, &secretTemplate)

		reconcileObject(t, secretTemplateReconciler, &secretTemplate)

		createdSecret := corev1.Secret{}
		err := k8sClient.Get(context.Background(), namespacedNameFor(&secretTemplate), &createdSecret)
		require.NoError(t, err)

		assert.Equal(t, []byte("value1"), createdSecret.Data["key1"])
	})
}

func Test_SecretTemplate_Embedded_Template(t *testing.T) {
	t.Run("reconciling secret template with input from another secret", func(t *testing.T) {
		configMap := ConfigMap("configmap1", map[string]string{
			"inputKey1": "value1",
		})

		secretTemplate := secretTemplate(
			"secretTemplate",
			"",
			map[string]client.Object{
				"map": &configMap,
			}, map[string]string{}, map[string]string{
				"embedded": "prefix-$( .map.data.inputKey1 )-suffix",
			})

		secretTemplateReconciler, k8sClient := importReconcilers(&configMap, &secretTemplate)

		reconcileObject(t, secretTemplateReconciler, &secretTemplate)

		createdSecret := corev1.Secret{}
		err := k8sClient.Get(context.Background(), namespacedNameFor(&secretTemplate), &createdSecret)
		require.NoError(t, err)

		assert.Equal(t, "prefix-value1-suffix", createdSecret.StringData["embedded"])
	})
}

func secretTemplate(name string, serviceAccount string, inputs map[string]client.Object, dataExpressions map[string]string, stringDataExpressions map[string]string) sg2v1alpha1.SecretTemplate {
	inputResources := []sg2v1alpha1.InputResource{}
	for key, obj := range inputs {
		inputResources = append(inputResources, sg2v1alpha1.InputResource{
			Name: key,
			Ref: sg2v1alpha1.InputResourceRef{
				APIVersion: obj.GetObjectKind().GroupVersionKind().GroupVersion().String(),
				Kind:       obj.GetObjectKind().GroupVersionKind().Kind,
				Name:       obj.GetName(),
			},
		})
	}
	return sg2v1alpha1.SecretTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "test",
		},
		Spec: sg2v1alpha1.SecretTemplateSpec{
			ServiceAccountName: serviceAccount,
			InputResources:     inputResources,
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

func ConfigMap(name string, data map[string]string) corev1.ConfigMap {
	return corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
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

	saLoader := generator.NewServiceAccountLoader(k8sClient)
	secretTemplateReconciler = generator.NewSecretTemplateReconciler(k8sClient, saLoader, testLogr)
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

func Test_SecretTemplate_Templating(t *testing.T) {
	type test struct {
		expression string
		expected   string
	}

	tests := []test{
		{expression: "static-value", expected: "static-value"},
		{expression: "$(.data.value)", expected: "{.data.value}"},
		{expression: "prefix-$(.data.value)-suffix", expected: "prefix-{.data.value}-suffix"},
		//failing
		// {expression: "$(.data.value)-middle-$(.data.value2)", expected: "{.data.value}-middle-{.data.value2}"},
	}

	for _, tc := range tests {
		expression := generator.TemplateSyntaxPath(tc.expression)
		result := expression.ToK8sJSONPath()
		if !reflect.DeepEqual(result, tc.expected) {
			t.Fatalf("expected: %v, got: %v", tc.expected, result)
		}
	}
}
