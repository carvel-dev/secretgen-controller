// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	sgv1alpha1 "github.com/vmware-tanzu/carvel-secretgen-controller/pkg/apis/secretgen/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name=Description,JSONPath=.status.friendlyDescription,description=Friendly description,type=string
// +kubebuilder:printcolumn:name=Age,JSONPath=.metadata.creationTimestamp,description=Time since creation,type=date
type SecretTemplate struct {
	metav1.TypeMeta `json:",inline"`

	// Standard object metadata; More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata.
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec SecretTemplateSpec `json:"spec"`
	// +optional
	Status SecretTemplateStatus `json:"status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type SecretTemplateList struct {
	metav1.TypeMeta `json:",inline"`

	// Standard list metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []SecretTemplate `json:"items"`
}

type SecretTemplateSpec struct {
	InputResources []InputResource `json:"inputResources"`

	// +optional
	JsonPathTemplate JsonPathTemplate `json:"template,omitempty"`
	// +optional
	YttTemplate YttTemplate `json:"ytt,omitempty"`

	// +optional
	ServiceAccountName string `json:"serviceAccountName,omitempty"`
}

type InputResource struct {
	Name string           `json:"name"`
	Ref  InputResourceRef `json:"ref"`
}

type InputResourceRef struct {
	ApiVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Name       string `json:"name"`
}

type JsonPathTemplate struct {
	// +optional
	StringData map[string]string `json:"stringData,omitempty"`
	// +optional
	Data map[string]string `json:"data,omitempty"`
}

type YttTemplate struct {
}

type SecretTemplateStatus struct {
	// +optional
	Binding Binding `json:"binding,omitempty"`

	sgv1alpha1.GenericStatus `json:",inline"`
	// +optional
	ObservedSecretResourceVersion string `json:"observedSecretResourceVersion,omitempty"`
}

type Binding struct {
	Name corev1.LocalObjectReference `json:"name"`
}

func (e SecretTemplate) Validate() error {
	var errs []error

	//TODO

	return combinedErrs("Validation errors", errs)
}
