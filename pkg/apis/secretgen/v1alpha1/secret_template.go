package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
)

type SecretTemplate struct {
	// TODO custom name is not supported as it makes "finding" secrets harder
	Metadata SecretTemplateMetadata `json:"metadata,omitempty"`
	Type     corev1.SecretType      `json:"type,omitempty"`
	Data     map[string]string      `json:"data,omitempty"`
}

type SecretTemplateMetadata struct {
	Annotations map[string]string `json:"annotations,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
}
