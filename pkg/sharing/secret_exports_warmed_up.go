// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package sharing

import (
	"sync"

	sg2v1alpha1 "github.com/vmware-tanzu/carvel-secretgen-controller/pkg/apis/secretgen2/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

// SecretExportsWarmedUp is a SecretExportsProvider
// that calls WarmUpFunc once on first access.
type SecretExportsWarmedUp struct {
	WarmUpFunc func()

	delegate   SecretExportsProvider
	warmUpOnce sync.Once
}

var _ SecretExportsProvider = &SecretExportsWarmedUp{}

// NewSecretExportsWarmedUp constructs new SecretExportsWarmedUp.
func NewSecretExportsWarmedUp(delegate SecretExportsProvider) *SecretExportsWarmedUp {
	return &SecretExportsWarmedUp{delegate: delegate, warmUpOnce: sync.Once{}}
}

// Export delegates.
func (se *SecretExportsWarmedUp) Export(export *sg2v1alpha1.SecretExport, secret *corev1.Secret) {
	se.delegate.Export(export, secret)
}

// Unexport delegates.
func (se *SecretExportsWarmedUp) Unexport(export *sg2v1alpha1.SecretExport) {
	se.delegate.Unexport(export)
}

// MatchedSecretsForImport warms up and then delegates.
func (se *SecretExportsWarmedUp) MatchedSecretsForImport(matcher SecretMatcher, nsIsExcludedFromWildcard NamespaceWildcardExclusionCheck) []*corev1.Secret {
	se.warmUpOnce.Do(se.WarmUpFunc)
	return se.delegate.MatchedSecretsForImport(matcher, nsIsExcludedFromWildcard)
}
