package v1alpha1

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type SecretRequest struct {
	metav1.TypeMeta `json:",inline"`

	// Standard object metadata; More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata.
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SecretRequestSpec   `json:"spec"`
	Status SecretRequestStatus `json:"status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type SecretRequestList struct {
	metav1.TypeMeta `json:",inline"`

	// Standard list metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []SecretRequest `json:"items"`
}

type SecretRequestSpec struct {
	FromNamespace string `json:"fromNamespace,omitempty"`
}

type SecretRequestStatus struct {
	GenericStatus `json:",inline"`
}

func (r SecretRequest) Validate() error {
	var errs []error

	if len(r.Spec.FromNamespace) == 0 {
		errs = append(errs, fmt.Errorf("Validating 'spec.fromNamespace': Expected to be non-empty"))
	}

	return combinedErrs("Validation errors", errs)
}
