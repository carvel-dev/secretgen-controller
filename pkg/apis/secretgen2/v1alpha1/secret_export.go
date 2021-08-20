// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type SecretExport struct {
	metav1.TypeMeta `json:",inline"`

	// Standard object metadata; More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata.
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SecretExportSpec   `json:"spec"`
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

type SecretExportSpec struct {
	ToNamespace  string   `json:"toNamespace,omitempty"`
	ToNamespaces []string `json:"toNamespaces,omitempty"`
}

type SecretExportStatus struct {
	GenericStatus                 `json:",inline"`
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

	if len(toNses) == 0 {
		errs = append(errs, fmt.Errorf("Expected to have at least one non-empty to namespace"))
	}
	for _, ns := range toNses {
		if len(ns) == 0 {
			errs = append(errs, fmt.Errorf("Expected to namespace to be non-empty"))
		}
	}

	return combinedErrs("Validation errors", errs)
}
