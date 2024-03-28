// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	sgv1alpha1 "github.com/vmware-tanzu/carvel-secretgen-controller/pkg/apis/secretgen/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SecretTemplate allows the construction of secrets using data that reside in other Kubernetes resources
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

// SecretTemplateList is a list of SecretTemplates
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type SecretTemplateList struct {
	metav1.TypeMeta `json:",inline"`

	// Standard list metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []SecretTemplate `json:"items"`
}

// SecretTemplateSpec contains spec information
type SecretTemplateSpec struct {
	// A list of input resources that are used to construct a new secret. Input Resources can refer to ANY Kubernetes API.
	// If loading more than Secrets types ensure that `.spec.ServiceAccountName` is set to an appropriate value.
	// Input resources are read in the order they are defined. An Input resource's name can be evaluated dynamically from data in a previously evaluated input resource.
	InputResources []InputResource `json:"inputResources"`

	// A JSONPath based template that can be used to create Secrets.
	// +optional
	JSONPathTemplate *JSONPathTemplate `json:"template,omitempty"`

	// The Service Account used to read InputResources. If not specified, only Secrets can be read as InputResources.
	// +optional
	ServiceAccountName string `json:"serviceAccountName,omitempty"`
}

// InputResource is references a single Kubernetes resource along with a identifying name
type InputResource struct {
	// The name of InputResource. This is used as the identifying name in templating to refer to this Input Resource.
	Name string `json:"name"`
	// The reference to the Input Resource
	Ref InputResourceRef `json:"ref"`
}

// InputResourceRef refers to a single Kubernetes resource
type InputResourceRef struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`

	// The name of the input resource. This field can itself contain JSONPATH syntax to load the name dynamically
	// from other input resources. For example this field could be set to a static value of "my-secret" or a dynamic valid of "$(.anotherinputresource.spec.name)".
	Name string `json:"name"`
}

// JSONPathTemplate contains templating information used to construct a new secret
type JSONPathTemplate struct {
	// StringData key and value. Where key is the Secret Key and the value can contain a JSONPATH syntax surrounded by $( ).
	// All InputResources are available via their identifying name.
	// For example:
	//   key1: static-text
	//   key2: $(.input1.spec.value1)
	//   key3: combined-$(.input2.status.value2)-$(.input2.status.value3)
	// +optional
	StringData map[string]string `json:"stringData,omitempty"`
	// Data key and value. Where key is the Secret Key and the value is a jsonpath surrounded by $( ). The fetched data MUST be base64 encoded.
	// All InputResources are available via their identifying name.
	// For example:
	//   key1: $(.secretinput1.data.value1)
	//   key2: $(.secretinput2.data.value2)
	// +optional
	Data map[string]string `json:"data,omitempty"`

	// Type is the type of Kubernetes Secret
	// +optional
	Type corev1.SecretType `json:"type,omitempty"`

	// Metadata contains metadata for the Secret
	// +optional
	Metadata SecretTemplateMetadata `json:"metadata,omitempty"`
}

// SecretTemplateMetadata allows the generated secret to contain metadata
type SecretTemplateMetadata struct {
	// Annotations to be placed on the generated secret
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`
	// Labels to be placed on the generated secret
	// +optional
	Labels map[string]string `json:"labels,omitempty"`
}

// SecretTemplateStatus contains status information
type SecretTemplateStatus struct {
	// +optional
	Secret corev1.LocalObjectReference `json:"secret,omitempty"`

	sgv1alpha1.GenericStatus `json:",inline"`
	// +optional
	ObservedSecretResourceVersion string `json:"observedSecretResourceVersion,omitempty"`
}
