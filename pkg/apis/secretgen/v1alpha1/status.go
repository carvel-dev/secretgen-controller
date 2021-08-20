// Copyright 2021 VMware, Inc.
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
