package sharing_test

import (
	"testing"

	"github.com/go-logr/logr"
	sgv1alpha1 "github.com/vmware-tanzu/carvel-secretgen-controller/pkg/apis/secretgen/v1alpha1"
	"github.com/vmware-tanzu/carvel-secretgen-controller/pkg/sharing"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"github.com/stretchr/testify/require"
)

func TestSecretExports(t *testing.T) {
	se := sharing.NewSecretExports(discardLogger{})

	// Namespace does not match
	secret1 := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "secret1", Namespace: "ns1"},
		Type: "Opaque",
		Data: map[string][]byte{"key1": []byte("val1")},
	}
	export1 := &sgv1alpha1.SecretExport{
		ObjectMeta: metav1.ObjectMeta{Name: "secret1", Namespace: "ns1"},
		Spec: sgv1alpha1.SecretExportSpec{
			ToNamespace: "wrong-ns",
		},
	}
	se.Export(export1, secret1)

	// Secret type does not match
	secret2 := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "secret2", Namespace: "ns1"},
		Type: "NotOpaque",
		Data: map[string][]byte{"key1": []byte("val1")},
	}
	export2 := &sgv1alpha1.SecretExport{
		ObjectMeta: metav1.ObjectMeta{Name: "secret2", Namespace: "ns1"},
		Spec: sgv1alpha1.SecretExportSpec{
			ToNamespace: "dst-ns",
		},
	}
	se.Export(export2, secret2)

	require.Equal(t, []*corev1.Secret(nil), se.MatchedSecretsForImport(sharing.SecretMatcher{
		Namespace: "dst-ns",
		SecretType: corev1.SecretType("Opaque"),
	}))

	// Everything matches
	secret3 := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "secret3", Namespace: "ns1"},
		Type: "Opaque",
		Data: map[string][]byte{"key1": []byte("val1")},
	}
	export3 := &sgv1alpha1.SecretExport{
		ObjectMeta: metav1.ObjectMeta{Name: "secret3", Namespace: "ns1"},
		Spec: sgv1alpha1.SecretExportSpec{
			ToNamespace: "dst-ns",
		},
	}
	se.Export(export3, secret3)

	require.Equal(t, []*corev1.Secret{secret3}, se.MatchedSecretsForImport(sharing.SecretMatcher{
		Namespace: "dst-ns",
		SecretType: corev1.SecretType("Opaque"),
	}))

	// Everything matches but from different namespace
	secret4 := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "secret4", Namespace: "ns2"},
		Type: "Opaque",
		Data: map[string][]byte{"key1": []byte("val1")},
	}
	export4 := &sgv1alpha1.SecretExport{
		ObjectMeta: metav1.ObjectMeta{Name: "secret4", Namespace: "ns2"},
		Spec: sgv1alpha1.SecretExportSpec{
			ToNamespace: "dst-ns",
		},
	}
	se.Export(export4, secret4)

	require.Equal(t,
		[]*corev1.Secret{secret3, secret4},
		se.MatchedSecretsForImport(sharing.SecretMatcher{
			Namespace: "dst-ns",
			SecretType: corev1.SecretType("Opaque"),
		}),
	)

	// Everything matches; exports to all namespaces
	secret5 := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "secret5", Namespace: "ns2"},
		Type: "Opaque",
		Data: map[string][]byte{"key1": []byte("val1")},
	}
	export5 := &sgv1alpha1.SecretExport{
		ObjectMeta: metav1.ObjectMeta{Name: "secret5", Namespace: "ns2"},
		Spec: sgv1alpha1.SecretExportSpec{
			ToNamespace: "*",
		},
	}
	se.Export(export5, secret5)

	// Everything matches; exports to multiple namespaces
	secret6 := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "secret6", Namespace: "ns2"},
		Type: "Opaque",
		Data: map[string][]byte{"key1": []byte("val1")},
	}
	export6 := &sgv1alpha1.SecretExport{
		ObjectMeta: metav1.ObjectMeta{Name: "secret6", Namespace: "ns2"},
		Spec: sgv1alpha1.SecretExportSpec{
			ToNamespace: "other-ns",
			ToNamespaces: []string{"dst-ns", "another-ns"},
		},
	}
	se.Export(export6, secret6)

	require.Equal(t,
		[]*corev1.Secret{secret3, secret4, secret5, secret6},
		se.MatchedSecretsForImport(sharing.SecretMatcher{
			Namespace: "dst-ns",
			SecretType: corev1.SecretType("Opaque"),
		}),
	)

	// No matches are produced when subject is offered for matching
	require.Equal(t, []*corev1.Secret(nil), se.MatchedSecretsForImport(sharing.SecretMatcher{
		Subject: "non-empty", // Currently not supported
		Namespace: "dst-ns",
		SecretType: corev1.SecretType("Opaque"),
	}))

	se.Unexport(export4)

	require.Equal(t,
		[]*corev1.Secret{secret3, secret5, secret6},
		se.MatchedSecretsForImport(sharing.SecretMatcher{
			Namespace: "dst-ns",
			SecretType: corev1.SecretType("Opaque"),
		}),
	)

	// Update secret export to no longer match namespace
	export6 = &sgv1alpha1.SecretExport{
		ObjectMeta: metav1.ObjectMeta{Name: "secret6", Namespace: "ns2"},
		Spec: sgv1alpha1.SecretExportSpec{
			ToNamespace: "other-ns",
			ToNamespaces: []string{"another-ns"},
		},
	}
	se.Export(export6, secret6)

	require.Equal(t,
		[]*corev1.Secret{secret3, secret5},
		se.MatchedSecretsForImport(sharing.SecretMatcher{
			Namespace: "dst-ns",
			SecretType: corev1.SecretType("Opaque"),
		}),
	)
}

type discardLogger struct{}

func (discardLogger) Info(msg string, keysAndValues ...interface{})             {}
func (discardLogger) Enabled() bool                                             { return true }
func (discardLogger) Error(err error, msg string, keysAndValues ...interface{}) {}
func (l discardLogger) V(level int) logr.InfoLogger                             { return l }
func (l discardLogger) WithValues(keysAndValues ...interface{}) logr.Logger     { return l }
func (l discardLogger) WithName(name string) logr.Logger                        { return l }
