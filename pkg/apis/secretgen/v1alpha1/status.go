// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

type GenericStatus struct {
	// +optional
	ObservedGeneration int64 `json:"observedGeneration"`
	// +optional
	Conditions []Condition `json:"conditions"`
	// +optional
	FriendlyDescription string `json:"friendlyDescription"`
}
