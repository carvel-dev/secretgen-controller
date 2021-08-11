// Copyright 2021 VMware, Inc.
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
type SSHKey struct {
	metav1.TypeMeta `json:",inline"`

	// Standard object metadata; More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata.
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SSHKeySpec   `json:"spec"`
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
	SecretTemplate *SecretTemplate `json:"secretTemplate,omitempty"`
}

type SSHKeyStatus struct {
	GenericStatus `json:",inline"`
}
