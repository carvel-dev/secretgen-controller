// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	RSAKeySecretPublicKeyKey  = "publicKey"
	RSAKeySecretPrivateKeyKey = "privateKey"

	RSAKeySecretDefaultType          = corev1.SecretTypeOpaque
	RSAKeySecretDefaultPublicKeyKey  = "pub.pem"
	RSAKeySecretDefaultPrivateKeyKey = "key.pem"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type RSAKey struct {
	metav1.TypeMeta `json:",inline"`

	// Standard object metadata; More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata.
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RSAKeySpec   `json:"spec"`
	Status RSAKeyStatus `json:"status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type RSAKeyList struct {
	metav1.TypeMeta `json:",inline"`

	// Standard list metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []RSAKey `json:"items"`
}

type RSAKeySpec struct {
	SecretTemplate *SecretTemplate `json:"secretTemplate,omitempty"`
}

type RSAKeyStatus struct {
	GenericStatus `json:",inline"`
}
