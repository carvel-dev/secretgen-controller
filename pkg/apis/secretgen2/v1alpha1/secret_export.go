// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"fmt"

	sgv1alpha1 "github.com/vmware-tanzu/carvel-secretgen-controller/pkg/apis/secretgen/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name=Description,JSONPath=.status.friendlyDescription,description=Friendly description,type=string
// +kubebuilder:printcolumn:name=Age,JSONPath=.metadata.creationTimestamp,description=Time since creation,type=date
type SecretExport struct {
	metav1.TypeMeta `json:",inline"`

	// Standard object metadata; More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata.
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec SecretExportSpec `json:"spec"`
	// +optional
	Status SecretExportStatus `json:"status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type SecretExportList struct {
	metav1.TypeMeta `json:",inline"`

	// Standard list metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []SecretExport `json:"items"`
}

// SelectorOperator is a part of SelectorMatchField
type SelectorOperator string

// SelectorOperator values
const (
	SelectorOperatorIn           SelectorOperator = "In"
	SelectorOperatorNotIn                         = "NotIn"
	SelectorOperatorExists                        = "Exists"
	SelectorOperatorDoesNotExist                  = "DoesNotExist"
)

// SelectorMatchField is a selector field to match against namespace definition
type SelectorMatchField struct {
	Key      string           `json:"key,omitempty"`
	Operator SelectorOperator `json:"operator,omitempty"`
	Values   []string         `json:"values,omitempty"`
}

type SecretExportSpec struct {
	// +optional
	ToNamespace string `json:"toNamespace,omitempty"`
	// +optional
	ToNamespaces []string `json:"toNamespaces,omitempty"`
	// +optional
	ToNamespacesSelector []SelectorMatchField `json:"dangerousToNamespacesSelector,omitempty"`
}

type SecretExportStatus struct {
	sgv1alpha1.GenericStatus `json:",inline"`
	// +optional
	ObservedSecretResourceVersion string `json:"observedSecretResourceVersion,omitempty"`
}

const (
	AllNamespaces = "*"
)

func (e SecretExport) StaticToNamespaces() []string {
	result := append([]string{}, e.Spec.ToNamespaces...)
	if len(e.Spec.ToNamespace) > 0 {
		result = append(result, e.Spec.ToNamespace)
	}
	return result
}

func (e SecretExport) Validate() error {
	var errs []error

	toNses := e.StaticToNamespaces()
	toSmf := e.Spec.ToNamespacesSelector

	if len(toNses) == 0 && len(toSmf) == 0 {
		errs = append(errs, fmt.Errorf("Expected to have at least one non-empty to namespace or to namespace annotation"))
	}
	for _, ns := range toNses {
		if len(ns) == 0 {
			errs = append(errs, fmt.Errorf("Expected to namespace to be non-empty"))
		}
	}
	for _, s := range toSmf {
		switch s.Operator {
		case SelectorOperatorIn, SelectorOperatorNotIn:
			if len(s.Values) == 0 {
				errs = append(errs, fmt.Errorf("Values must be specified when `operator` is 'In' or 'NotIn'"))
			}
		case SelectorOperatorExists, SelectorOperatorDoesNotExist:
			if len(s.Values) > 0 {
				errs = append(errs, fmt.Errorf("Values may not be specified when `operator` is 'Exists' or 'DoesNotExist'"))
			}
		}
	}

	return combinedErrs("Validation errors", errs)
}
