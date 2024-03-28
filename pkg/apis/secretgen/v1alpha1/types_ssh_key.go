// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	SSHKeySecretPrivateKeyKey    = "privateKey"
	SSHKeySecretAuthorizedKeyKey = "authorizedKey"

	SSHKeySecretDefaultType             = corev1.SecretTypeSSHAuth
	SSHKeySecretDefaultPrivateKeyKey    = corev1.SSHAuthPrivateKey
	SSHKeySecretDefaultAuthorizedKeyKey = "ssh-authorizedkey"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name=Description,JSONPath=.status.friendlyDescription,description=Friendly description,type=string
// +kubebuilder:printcolumn:name=Age,JSONPath=.metadata.creationTimestamp,description=Time since creation,type=date
type SSHKey struct {
	metav1.TypeMeta `json:",inline"`

	// Standard object metadata; More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata.
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// +optional
	Spec SSHKeySpec `json:"spec"`
	// +optional
	Status SSHKeyStatus `json:"status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type SSHKeyList struct {
	metav1.TypeMeta `json:",inline"`

	// Standard list metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []SSHKey `json:"items"`
}

type SSHKeySpec struct {
	// +optional
	SecretTemplate *SecretTemplate `json:"secretTemplate,omitempty"`
}

type SSHKeyStatus struct {
	GenericStatus `json:",inline"`
}
