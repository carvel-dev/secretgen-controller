// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

type GenericStatus struct {
	ObservedGeneration  int64       `json:"observedGeneration"`
	Conditions          []Condition `json:"conditions"`
	FriendlyDescription string      `json:"friendlyDescription"`
}
