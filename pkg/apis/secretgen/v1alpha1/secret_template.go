package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
)

type SecretTemplate struct {
	// TODO custom name is not supported as it makes "finding" secrets harder
	Metadata   SecretTemplateMetadata `json:"metadata,omitempty"`
	Type       corev1.SecretType      `json:"type,omitempty"`
	StringData map[string]string      `json:"stringData,omitempty"`
}

type SecretTemplateMetadata struct {
	Annotations map[string]string `json:"annotations,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
}
