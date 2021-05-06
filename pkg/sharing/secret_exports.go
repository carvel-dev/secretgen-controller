package sharing

import (
	"sync"

	"github.com/go-logr/logr"
	sgv1alpha1 "github.com/vmware-tanzu/carvel-secretgen-controller/pkg/apis/secretgen/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

type SecretExports struct {
	log logr.Logger

	exportedSecretsLock sync.RWMutex
	exportedSecrets     map[string]exportedSecret
}

type exportedSecret struct {
	*sgv1alpha1.SecretExport
	*corev1.Secret
}

func (p exportedSecret) Key() string {
	if p.SecretExport.Namespace == "" {
		panic("Internal inconsistency: missing namespace")
	}
	if p.SecretExport.Name == "" {
		panic("Internal inconsistency: missing name")
	}
	return p.SecretExport.Namespace + "/" + p.SecretExport.Name
}

func (p exportedSecret) Matches(matcher SecretMatcher) bool {
	if matcher.Subject != "" {
		// TODO we currently do not match by subject
		return false
	}
	if matcher.SecretType != p.Secret.Type {
		return false
	}
	if !p.matchesNamespace(matcher.Namespace) {
		return false
	}
	return true
}

func (p exportedSecret) matchesNamespace(nsToMatch string) bool {
	for _, ns := range p.SecretExport.StaticToNamespaces() {
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
