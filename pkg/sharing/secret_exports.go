package sharing

import (
	"sync"

	"github.com/go-logr/logr"
	sgv1alpha1 "github.com/vmware-tanzu/carvel-secretgen-controller/pkg/apis/secretgen/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

// SecretExports is an in-memory cache of exported secrets.
// It can be asked to return secrets that match specific criteria for importing.
// (SecretExports is used by SecretExportReconciler to export/unexport secrets;
// SecretExports is used by SecretReconciler to determine imported secrets.)
type SecretExports struct {
	log logr.Logger

	exportedSecretsLock sync.RWMutex
	exportedSecrets     map[string]exportedSecret
}

type exportedSecret struct {
	*sgv1alpha1.SecretExport
	*corev1.Secret
}

func (es exportedSecret) Key() string {
	if es.SecretExport.Namespace == "" {
		panic("Internal inconsistency: missing namespace")
	}
	if es.SecretExport.Name == "" {
		panic("Internal inconsistency: missing name")
	}
	return es.SecretExport.Namespace + "/" + es.SecretExport.Name
}

func (es exportedSecret) Matches(matcher SecretMatcher) bool {
	if matcher.Subject != "" {
		// TODO we currently do not match by subject
		return false
	}
	if matcher.SecretType != es.Secret.Type {
		return false
	}
	if !es.matchesNamespace(matcher.Namespace) {
		return false
	}
	return true
}

func (es exportedSecret) matchesNamespace(nsToMatch string) bool {
	for _, ns := range es.SecretExport.StaticToNamespaces() {
		if ns == sgv1alpha1.AllNamespaces || ns == nsToMatch {
			return true
		}
	}
	return false
}

func NewSecretExports(log logr.Logger) *SecretExports {
	return &SecretExports{log: log, exportedSecrets: map[string]exportedSecret{}}
}

func (se *SecretExports) Export(export *sgv1alpha1.SecretExport, secret *corev1.Secret) {
	if export.Namespace != secret.Namespace || export.Name != secret.Name {
		panic("Internal inconsistency: export and secret names do not match")
	}

	se.exportedSecretsLock.Lock()
	defer se.exportedSecretsLock.Unlock()

	exportedSec := exportedSecret{export, secret}
	se.exportedSecrets[exportedSec.Key()] = exportedSec
}

func (se *SecretExports) Unexport(export *sgv1alpha1.SecretExport) {
	se.exportedSecretsLock.Lock()
	defer se.exportedSecretsLock.Unlock()

	delete(se.exportedSecrets, exportedSecret{export, nil}.Key())
}

type SecretMatcher struct {
	Namespace  string
	Subject    string
	SecretType corev1.SecretType
}

// MatchedSecretsForImport filters secrets export cache by the given criteria.
func (se *SecretExports) MatchedSecretsForImport(matcher SecretMatcher) []*corev1.Secret {
	se.exportedSecretsLock.RLock()
	defer se.exportedSecretsLock.RUnlock()

	var result []*corev1.Secret

	for _, exportedSec := range se.exportedSecrets {
		if exportedSec.Matches(matcher) {
			result = append(result, exportedSec.Secret)
		}
	}

	return result
}
