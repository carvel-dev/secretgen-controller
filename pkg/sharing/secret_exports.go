// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package sharing

import (
	"sync"
	"sort"

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

func NewSecretExports(log logr.Logger) *SecretExports {
	return &SecretExports{log: log, exportedSecrets: map[string]exportedSecret{}}
}

func (se *SecretExports) Export(export *sgv1alpha1.SecretExport, secret *corev1.Secret) {
	if secret == nil {
		panic("Internal inconsistency: expected non-nil secret")
	}
	exportedSec := newExportedSecret(export, secret)

	se.exportedSecretsLock.Lock()
	defer se.exportedSecretsLock.Unlock()

	se.exportedSecrets[exportedSec.Key()] = exportedSec
}

// Unexport deletes the in-memory representation (cached)
// of both the SecretExport and underlying Secret.
func (se *SecretExports) Unexport(export *sgv1alpha1.SecretExport) {
	exportedSec := newExportedSecret(export, nil)

	se.exportedSecretsLock.Lock()
	defer se.exportedSecretsLock.Unlock()

	delete(se.exportedSecrets, exportedSec.Key())
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
			result = append(result, exportedSec.Secret())
		}
	}

	// Return determinsticly ordered result
	sort.Slice(result, func(i, j int) bool {
		// First by namespace, then by name
		if result[i].Namespace != result[j].Namespace {
			return result[i].Namespace < result[j].Namespace
		}
		return result[i].Name < result[j].Name
	})

	return result
}

// exportedSecret used for keeping track export->secret pair.
type exportedSecret struct {
	export *sgv1alpha1.SecretExport
	secret *corev1.Secret
}

func newExportedSecret(export *sgv1alpha1.SecretExport, secret *corev1.Secret) exportedSecret {
	if export == nil {
		panic("Internal inconsistency: nil export")
	}
	if export.Namespace == "" {
		panic("Internal inconsistency: missing export namespace")
	}
	if export.Name == "" {
		panic("Internal inconsistency: missing export name")
	}
	if secret != nil {
		if export.Namespace != secret.Namespace || export.Name != secret.Name {
			panic("Internal inconsistency: export and secret names do not match")
		}
		secret = secret.DeepCopy()
	}
	return exportedSecret{export.DeepCopy(), secret}
}

func (es exportedSecret) Key() string {
	return es.export.Namespace + "/" + es.export.Name
}

func (es exportedSecret) Secret() *corev1.Secret {
	return es.secret.DeepCopy()
}

func (es exportedSecret) Matches(matcher SecretMatcher) bool {
	if matcher.Subject != "" {
		// TODO we currently do not match by subject
		return false
	}
	if matcher.SecretType != es.secret.Type {
		return false
	}
	if !es.matchesNamespace(matcher.Namespace) {
		return false
	}
	return true
}

func (es exportedSecret) matchesNamespace(nsToMatch string) bool {
	for _, ns := range es.export.StaticToNamespaces() {
		if ns == sgv1alpha1.AllNamespaces || ns == nsToMatch {
			return true
		}
	}
	return false
}
