// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
)

type ConditionType string

const (
	Reconciling        ConditionType = "Reconciling"
	ReconcileFailed    ConditionType = "ReconcileFailed"
	ReconcileSucceeded ConditionType = "ReconcileSucceeded"

	// Invalid indicates that CRD's spec is not of valid form.
	Invalid ConditionType = "Invalid"
)

type Condition struct {
	// +optional
	Type ConditionType `json:"type"`
	// +optional
	Status corev1.ConditionStatus `json:"status"`
	// Unique, this should be a short, machine understandable string that gives the reason
	// for condition's last transition. If it reports "ResizeStarted" that means the underlying
	// persistent volume is being resized.
	// +optional
	Reason string `json:"reason,omitempty"`
	// Human-readable message indicating details about last transition.
	// +optional
	Message string `json:"message,omitempty"`
}
