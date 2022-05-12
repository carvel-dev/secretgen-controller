// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package generator_test

import (
	"context"
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
	type test struct {
		name            string
		template        sg2v1alpha1.SecretTemplate
		existingObjects []client.Object
		expectedData    map[string]string
	}

	tests := []test{
		{
			name: "reconciling secret template with input from another secret",
			template: sg2v1alpha1.SecretTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secretTemplate",
					Namespace: "test",
				},
				Spec: sg2v1alpha1.SecretTemplateSpec{
					InputResources: []sg2v1alpha1.InputResource{{
						Name: "creds",
						Ref: sg2v1alpha1.InputResourceRef{
							APIVersion: "v1",
							Kind:       "Secret",
							Name:       "existingSecret",
						},
					}},
					JSONPathTemplate: sg2v1alpha1.JSONPathTemplate{
						Data: map[string]string{
							"key1": "$( .creds.data.inputKey1 )",
							"key2": "$( .creds.data.inputKey2 )",
						},
						StringData: map[string]string{
							"key3": "value3",
						},
					},
				},
			},
			existingObjects: []client.Object{
				secret("existingSecret", map[string]string{
					"inputKey1": "value1",
					"inputKey2": "value2"}),
			},
			expectedData: map[string]string{
				"key1": "value1",
				"key2": "value2",
				"key3": "value3",
			},
		},
		{
			name: "reconciling secret template with input from two inputs with dynamic inputname",
			template: sg2v1alpha1.SecretTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secretTemplate",
					Namespace: "test",
				},
				Spec: sg2v1alpha1.SecretTemplateSpec{
					InputResources: []sg2v1alpha1.InputResource{{
						Name: "first",
						Ref: sg2v1alpha1.InputResourceRef{
							APIVersion: "v1",
							Kind:       "ConfigMap",
							Name:       "first",
						},
					},
						{
							Name: "creds",
							Ref: sg2v1alpha1.InputResourceRef{
								APIVersion: "v1",
								Kind:       "Secret",
								Name:       "$( .first.data.secretName )",
							},
						}},
					JSONPathTemplate: sg2v1alpha1.JSONPathTemplate{
						Data: map[string]string{
							"key1": "$( .creds.data.inputKey1 )",
						},
					},
				},
			},
			existingObjects: []client.Object{
				configMap("first", map[string]string{
					"secretName": "dynamic-secret-name",
				}),
				secret("dynamic-secret-name", map[string]string{
					"inputKey1": "value1",
				}),
			},
			expectedData: map[string]string{
				"key1": "value1",
			},
		},
		{
			name: "reconciling secret template with embedded stringData template from configmap",
			template: sg2v1alpha1.SecretTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secretTemplate",
					Namespace: "test",
				},
				Spec: sg2v1alpha1.SecretTemplateSpec{
					InputResources: []sg2v1alpha1.InputResource{{
						Name: "map",
						Ref: sg2v1alpha1.InputResourceRef{
							APIVersion: "v1",
							Kind:       "ConfigMap",
							Name:       "existingcfgmap",
						},
					}},
					JSONPathTemplate: sg2v1alpha1.JSONPathTemplate{
						StringData: map[string]string{
							"key1": "prefix-$(.map.data.inputKey1)-suffix",
						},
					},
				},
			},
			existingObjects: []client.Object{
				configMap("existingcfgmap", map[string]string{
					"inputKey1": "value1",
				}),
			},
			expectedData: map[string]string{
				"key1": "prefix-value1-suffix",
			},
		},
		// Failing (maybe this should only be supported in ytt?)
		//
		// {
		// 	name: "reconciling secret template with embedded stringData template from secret",
		// 	template: sg2v1alpha1.SecretTemplate{
		// 		ObjectMeta: metav1.ObjectMeta{
		// 			Name:      "secretTemplate",
		// 			Namespace: "test",
		// 		},
		// 		Spec: sg2v1alpha1.SecretTemplateSpec{
		// 			InputResources: []sg2v1alpha1.InputResource{{
		// 				Name: "creds",
		// 				Ref: sg2v1alpha1.InputResourceRef{
		// 					APIVersion: "v1",
		// 					Kind:       "Secret",
		// 					Name:       "existingsecret",
		// 				},
		// 			}},
		// 			JSONPathTemplate: sg2v1alpha1.JSONPathTemplate{
		// 				StringData: map[string]string{
		// 					"key1": "prefix-$(.creds.inputKey1)-suffix",
		// 				},
		// 			},
		// 		},
		// 	},
		// 	objects: []client.Object{
		// 		ConfigMap("existingsecret", map[string]string{
		// 			"inputKey1": "value1",
		// 		}),
		// 	},
		// 	expectedData: map[string]string{
		// 		"key1": "prefix-value1-suffix",
		// 	},
		// },
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			allObjects := append(tc.existingObjects, &tc.template)
			secretTemplateReconciler, k8sClient := importReconcilers(allObjects...)

			reconcileObject(t, secretTemplateReconciler, &tc.template)

			createdSecret := corev1.Secret{}
			err := k8sClient.Get(context.Background(), namespacedNameFor(&tc.template), &createdSecret)
			require.NoError(t, err)

			actual := map[string]string{}
			for key, value := range createdSecret.StringData {
				actual[key] = value
			}
			for key, value := range createdSecret.Data {
				actual[key] = string(value)
			}

			for key, value := range tc.expectedData {
				assert.Equal(t, value, actual[key])
			}
		})
	}
}

func secret(name string, stringData map[string]string) *corev1.Secret {
	data := map[string][]byte{}

	for key, val := range stringData {
		data[key] = []byte(val)
	}

	return &corev1.Secret{
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

func configMap(name string, data map[string]string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
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
