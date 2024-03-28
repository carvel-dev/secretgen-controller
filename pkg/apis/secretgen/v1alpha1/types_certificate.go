// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	CertificateSecretCertificateKey = "certificate"
	CertificateSecretPrivateKeyKey  = "privateKey"

	CertificateSecretDefaultType           = corev1.SecretTypeOpaque
	CertificateSecretDefaultCertificateKey = "crt.pem"
	CertificateSecretDefaultPrivateKeyKey  = "key.pem"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name=Description,JSONPath=.status.friendlyDescription,description=Friendly description,type=string
// +kubebuilder:printcolumn:name=Age,JSONPath=.metadata.creationTimestamp,description=Time since creation,type=date
type Certificate struct {
	metav1.TypeMeta `json:",inline"`

	// Standard object metadata; More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata.
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// +optional
	Spec CertificateSpec `json:"spec"`
	// +optional
	Status CertificateStatus `json:"status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type CertificateList struct {
	metav1.TypeMeta `json:",inline"`

	// Standard list metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Certificate `json:"items"`
}

type CertificateSpec struct {
	// +optional
	CARef *corev1.LocalObjectReference `json:"caRef,omitempty"`
	// +optional
	IsCA bool `json:"isCA,omitempty"`

	// +optional
	CommonName string `json:"commonName,omitempty"`
	// +optional
	Organization string `json:"organization,omitempty"`
	// +optional
	AlternativeNames []string `json:"alternativeNames,omitempty"`
	// +optional
	ExtendedKeyUsage []string `json:"extendedKeyUsage,omitempty"`
	// +optional
	Duration int64 `json:"duration,omitempty"`

	// +optional
	SecretTemplate *SecretTemplate `json:"secretTemplate,omitempty"`
}

type CertificateStatus struct {
	GenericStatus `json:",inline"`
}
