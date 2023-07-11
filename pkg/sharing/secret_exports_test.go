// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package sharing_test

import (
	"fmt"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"
	sg2v1alpha1 "github.com/vmware-tanzu/carvel-secretgen-controller/pkg/apis/secretgen2/v1alpha1"
	"github.com/vmware-tanzu/carvel-secretgen-controller/pkg/sharing"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakeClient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestSecretExports(t *testing.T) {

	t.Run("matching", func(t *testing.T) {
		// Namespace does not match
		secret1 := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "secret1", Namespace: "ns1"},
			Type:       "Opaque",
			Data:       map[string][]byte{"key1": []byte("val1")},
		}
		export1 := &sg2v1alpha1.SecretExport{
			ObjectMeta: metav1.ObjectMeta{Name: "secret1", Namespace: "ns1"},
			Spec: sg2v1alpha1.SecretExportSpec{
				ToNamespace: "wrong-ns",
			},
		}

		// Secret type does not match
		secret2 := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "secret2", Namespace: "ns1"},
			Type:       "NotOpaque",
			Data:       map[string][]byte{"key1": []byte("val1")},
		}
		export2 := &sg2v1alpha1.SecretExport{
			ObjectMeta: metav1.ObjectMeta{Name: "secret2", Namespace: "ns1"},
			Spec: sg2v1alpha1.SecretExportSpec{
				ToNamespace: "dst-ns",
			},
		}

		k8sClient := fakeClient.NewFakeClient(secret1, secret2, export1, export2)
		se := sharing.NewSecretExports(k8sClient, logr.Discard())

		se.Export(export1, secret1)

		se.Export(export2, secret2)

		nsCheck := func(string) bool { return false }
		require.Equal(t, []*corev1.Secret(nil),
			se.MatchedSecretsForImport(sharing.SecretMatcher{
				ToNamespace: "dst-ns",
				SecretType:  corev1.SecretType("Opaque"),
			}, nsCheck))

		// Everything matches
		secret3 := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "secret3", Namespace: "ns1"},
			Type:       "Opaque",
			Data:       map[string][]byte{"key1": []byte("val1")},
		}
		export3 := &sg2v1alpha1.SecretExport{
			ObjectMeta: metav1.ObjectMeta{Name: "secret3", Namespace: "ns1"},
			Spec: sg2v1alpha1.SecretExportSpec{
				ToNamespace: "dst-ns",
			},
		}
		se.Export(export3, secret3)

		require.Equal(t, []*corev1.Secret{secret3}, se.MatchedSecretsForImport(sharing.SecretMatcher{
			ToNamespace: "dst-ns",
			SecretType:  corev1.SecretType("Opaque"),
		}, nsCheck))

		// Everything matches but from different namespace
		secret4 := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "secret4", Namespace: "ns2"},
			Type:       "Opaque",
			Data:       map[string][]byte{"key1": []byte("val1")},
		}
		export4 := &sg2v1alpha1.SecretExport{
			ObjectMeta: metav1.ObjectMeta{Name: "secret4", Namespace: "ns2"},
			Spec: sg2v1alpha1.SecretExportSpec{
				ToNamespace: "dst-ns",
			},
		}
		se.Export(export4, secret4)

		require.Equal(t,
			[]*corev1.Secret{secret3, secret4},
			se.MatchedSecretsForImport(sharing.SecretMatcher{
				ToNamespace: "dst-ns",
				SecretType:  corev1.SecretType("Opaque"),
			}, nsCheck),
		)

		// Everything matches; exports to all namespaces
		secret5 := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "secret5", Namespace: "ns2"},
			Type:       "Opaque",
			Data:       map[string][]byte{"key1": []byte("val1")},
		}
		export5 := &sg2v1alpha1.SecretExport{
			ObjectMeta: metav1.ObjectMeta{Name: "secret5", Namespace: "ns2"},
			Spec: sg2v1alpha1.SecretExportSpec{
				ToNamespace: "*",
			},
		}
		se.Export(export5, secret5)

		// Everything matches; exports to multiple namespaces
		secret6 := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "secret6", Namespace: "ns2"},
			Type:       "Opaque",
			Data:       map[string][]byte{"key1": []byte("val1")},
		}
		export6 := &sg2v1alpha1.SecretExport{
			ObjectMeta: metav1.ObjectMeta{Name: "secret6", Namespace: "ns2"},
			Spec: sg2v1alpha1.SecretExportSpec{
				ToNamespace:  "other-ns",
				ToNamespaces: []string{"dst-ns", "another-ns"},
			},
		}
		se.Export(export6, secret6)

		require.Equal(t,
			[]*corev1.Secret{secret5, secret3, secret4, secret6},
			se.MatchedSecretsForImport(sharing.SecretMatcher{
				ToNamespace: "dst-ns",
				SecretType:  corev1.SecretType("Opaque"),
			}, nsCheck),
		)

		// No matches are produced when subject is offered for matching
		require.Equal(t, []*corev1.Secret(nil), se.MatchedSecretsForImport(sharing.SecretMatcher{
			Subject:     "non-empty", // Currently not supported
			ToNamespace: "dst-ns",
			SecretType:  corev1.SecretType("Opaque"),
		}, nsCheck))

		se.Unexport(export4)

		require.Equal(t,
			[]*corev1.Secret{secret5, secret3, secret6},
			se.MatchedSecretsForImport(sharing.SecretMatcher{
				ToNamespace: "dst-ns",
				SecretType:  corev1.SecretType("Opaque"),
			}, nsCheck),
		)

		// Update secret export to no longer match namespace
		export6 = &sg2v1alpha1.SecretExport{
			ObjectMeta: metav1.ObjectMeta{Name: "secret6", Namespace: "ns2"},
			Spec: sg2v1alpha1.SecretExportSpec{
				ToNamespace:  "other-ns",
				ToNamespaces: []string{"another-ns"},
			},
		}
		se.Export(export6, secret6)

		require.Equal(t,
			[]*corev1.Secret{secret5, secret3},
			se.MatchedSecretsForImport(sharing.SecretMatcher{
				ToNamespace: "dst-ns",
				SecretType:  corev1.SecretType("Opaque"),
			}, nsCheck),
		)

		// now if the nsCheck returns true for a ns under a * match it shouldn't share that secret
		require.Equal(t,
			[]*corev1.Secret{secret3},
			se.MatchedSecretsForImport(sharing.SecretMatcher{
				ToNamespace: "dst-ns",
				SecretType:  corev1.SecretType("Opaque"),
			}, func(ns string) bool { return true }))
	})

	t.Run("returns secrets in specific order (last secret is most preferred)", func(t *testing.T) {

		// higher weight, but name comes earlier
		secret1 := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "secret1-highest-weight", Namespace: "ns1"},
			Type:       "Opaque",
		}
		export1 := &sg2v1alpha1.SecretExport{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "secret1-highest-weight",
				Namespace: "ns1",
				Annotations: map[string]string{
					"secretgen.carvel.dev/weight": "10.0",
				},
			},
			Spec: sg2v1alpha1.SecretExportSpec{ToNamespace: "*"},
		}

		// higher weight, but name comes later
		secret2 := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "secret1a-highest-weight", Namespace: "ns1"},
			Type:       "Opaque",
		}
		export2 := &sg2v1alpha1.SecretExport{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "secret1a-highest-weight",
				Namespace: "ns1",
				Annotations: map[string]string{
					"secretgen.carvel.dev/weight": "10.0",
				},
			},
			Spec: sg2v1alpha1.SecretExportSpec{ToNamespace: "*"},
		}

		secret3 := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "secret2-low-weight", Namespace: "ns1"},
			Type:       "Opaque",
		}
		export3 := &sg2v1alpha1.SecretExport{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "secret2-low-weight",
				Namespace: "ns1",
				Annotations: map[string]string{
					"secretgen.carvel.dev/weight": "1.0",
				},
			},
			Spec: sg2v1alpha1.SecretExportSpec{ToNamespace: "*"},
		}

		// Wildcard ns match
		secret4 := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "secret3-diff-ns-wildcard-ns", Namespace: "ns1"},
			Type:       "Opaque",
		}
		export4 := &sg2v1alpha1.SecretExport{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "secret3-diff-ns-wildcard-ns",
				Namespace: "ns1",
			},
			Spec: sg2v1alpha1.SecretExportSpec{ToNamespace: "*"},
		}

		// Specific ns match (even though there is a wildcard as well)
		secret5 := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "secret4-diff-ns-specific-ns", Namespace: "ns1"},
			Type:       "Opaque",
		}
		export5 := &sg2v1alpha1.SecretExport{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "secret4-diff-ns-specific-ns",
				Namespace: "ns1",
			},
			Spec: sg2v1alpha1.SecretExportSpec{ToNamespace: "dst-ns", ToNamespaces: []string{"*"}},
		}

		k8sClient := fakeClient.NewFakeClient(secret1, secret2, export1, export2)
		se := sharing.NewSecretExports(k8sClient, logr.Discard())

		// Wildcard ns match in same namespace
		secret6 := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "secret5-same-ns-wildcard-ns", Namespace: "dst-ns"},
			Type:       "Opaque",
		}
		export6 := &sg2v1alpha1.SecretExport{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "secret5-same-ns-wildcard-ns",
				Namespace: "dst-ns",
			},
			Spec: sg2v1alpha1.SecretExportSpec{ToNamespace: "*"},
		}

		// Specific ns match (even though there is a wildcard as well)
		secret7 := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "secret6-same-ns-specific-ns", Namespace: "dst-ns"},
			Type:       "Opaque",
		}
		export7 := &sg2v1alpha1.SecretExport{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "secret6-same-ns-specific-ns",
				Namespace: "dst-ns",
			},
			Spec: sg2v1alpha1.SecretExportSpec{ToNamespace: "dst-ns", ToNamespaces: []string{"*"}},
		}

		se.Export(export1, secret1)

		se.Export(export2, secret2)

		se.Export(export3, secret3)

		se.Export(export4, secret4)

		se.Export(export5, secret5)

		se.Export(export6, secret6)

		se.Export(export7, secret7)

		result := se.MatchedSecretsForImport(sharing.SecretMatcher{
			ToNamespace: "dst-ns",
			SecretType:  corev1.SecretType("Opaque"),
		}, func(string) bool { return false })

		// Check based on metas since assertion diff will be more readable
		var actualMetas []string
		for _, secret := range result {
			actualMetas = append(actualMetas, fmt.Sprintf("%s/%s", secret.Namespace, secret.Name))
		}

		require.Equal(t, []string{
			"ns1/secret3-diff-ns-wildcard-ns",
			"ns1/secret4-diff-ns-specific-ns",
			"dst-ns/secret5-same-ns-wildcard-ns",
			"dst-ns/secret6-same-ns-specific-ns",
			"ns1/secret2-low-weight",
			"ns1/secret1-highest-weight",
			"ns1/secret1a-highest-weight",
		}, actualMetas)
	})
}
