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
type SecretImport struct {
	metav1.TypeMeta `json:",inline"`

	// Standard object metadata; More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata.
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec SecretImportSpec `json:"spec"`
	// +optional
	Status SecretImportStatus `json:"status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type SecretImportList struct {
	metav1.TypeMeta `json:",inline"`

	// Standard list metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []SecretImport `json:"items"`
}

type SecretImportSpec struct {
	// +optional
	FromNamespace string `json:"fromNamespace,omitempty"`
}

type SecretImportStatus struct {
	sgv1alpha1.GenericStatus `json:",inline"`
}

func (r SecretImport) Validate() error {
	var errs []error

	if len(r.Spec.FromNamespace) == 0 {
		errs = append(errs, fmt.Errorf("Validating 'spec.fromNamespace': Expected to be non-empty"))
	}

	return combinedErrs("Validation errors", errs)
}
