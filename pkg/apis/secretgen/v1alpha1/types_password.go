// Copyright 2024 The Carvel Authors.
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
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name=Description,JSONPath=.status.friendlyDescription,description=Friendly description,type=string
// +kubebuilder:printcolumn:name=Age,JSONPath=.metadata.creationTimestamp,description=Time since creation,type=date
type Password struct {
	metav1.TypeMeta `json:",inline"`

	// Standard object metadata; More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata.
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// +optional
	Spec PasswordSpec `json:"spec"`
	// +optional
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
	// +optional
	// +kubebuilder:default:=40
	Length int `json:"length"`

	// +optional
	// +kubebuilder:default:=0
	Digits int `json:"digits"`

	// +optional
	// +kubebuilder:default:=0
	Symbols int `json:"symbols"`

	// +optional
	// +kubebuilder:default:=0
	UppercaseLetters int `json:"uppercaseLetters"`

	// +optional
	// +kubebuilder:default:=0
	LowercaseLetters int `json:"lowercaseLetters"`

	// +optional
	// +kubebuilder:default:="!@#$%&*;.:"
	SymbolCharSet string `json:"symbolCharSet"`

	// +optional
	SecretTemplate *SecretTemplate `json:"secretTemplate,omitempty"`
}

type PasswordStatus struct {
	GenericStatus `json:",inline"`
}
