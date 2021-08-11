// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	PasswordSecretKey = "value"

	PasswordSecretDefaultType = corev1.SecretTypeBasicAuth
	PasswordSecretDefaultKey  = corev1.BasicAuthPasswordKey
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type Password struct {
	metav1.TypeMeta `json:",inline"`

	// Standard object metadata; More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata.
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PasswordSpec   `json:"spec"`
	Status PasswordStatus `json:"status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type PasswordList struct {
	metav1.TypeMeta `json:",inline"`

	// Standard list metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Password `json:"items"`
}

type PasswordSpec struct {
	Length int `json:"length"`

	SecretTemplate *SecretTemplate `json:"secretTemplate,omitempty"`
}

type PasswordStatus struct {
	GenericStatus `json:",inline"`
}
