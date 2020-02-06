package v1alpha1

type GenericStatus struct {
	ObservedGeneration  int64       `json:"observedGeneration"`
	Conditions          []Condition `json:"conditions"`
	FriendlyDescription string      `json:"friendlyDescription"`
}
