package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	CertificateSecretType           = corev1.SecretTypeOpaque
	CertificateSecretCertificateKey = "crt.pem"
	CertificateSecretPrivateKeyKey  = "key.pem"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type Certificate struct {
	metav1.TypeMeta `json:",inline"`

	// Standard object metadata; More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata.
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CertificateSpec   `json:"spec"`
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
	// TODO should we support cross namespace references?
	CARef *corev1.LocalObjectReference `json:"caRef,omitempty"`
	IsCA  bool                         `json:"isCA,omitempty"`

	CommonName       string   `json:"commonName,omitempty"`
	Organization     string   `json:"organization,omitempty"`
	AlternativeNames []string `json:"alternativeNames,omitempty"`
	ExtendedKeyUsage []string `json:"extendedKeyUsage,omitempty"`
	Duration         int64    `json:"duration,omitempty"`
}

type CertificateStatus struct {
	GenericStatus `json:",inline"`
}
