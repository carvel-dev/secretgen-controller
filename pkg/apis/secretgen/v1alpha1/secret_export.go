package v1alpha1

import (
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

func (e SecretExport) StaticToNamespaces() []string {
	result := append([]string{}, e.Spec.ToNamespaces...)
	if len(e.Spec.ToNamespace) > 0 {
		result = append(result, e.Spec.ToNamespace)
	}
	return result
}
