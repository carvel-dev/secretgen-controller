// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
)

type SecretTemplate struct {
	// +optional
	Metadata SecretTemplateMetadata `json:"metadata,omitempty"`
	// +optional
	Type corev1.SecretType `json:"type,omitempty"`
	// +optional
	StringData map[string]string `json:"stringData,omitempty"`
}

type SecretTemplateMetadata struct {
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`
	// +optional
	Labels map[string]string `json:"labels,omitempty"`
}
