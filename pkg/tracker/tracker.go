// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

// Package tracker allows "tracking" resources to monitor "tracked" resources.
// All "tracking" resources can then be found for a given "tracked" resource.
// Tracker is thread-safe.
package tracker

import (
	"sync"

	"k8s.io/apimachinery/pkg/types"
)

// Tracker allows "tracking" resources to monitor "tracked" resources.
// All "tracking" resources can then be found for a given "tracked" resource.
// Tracker is thread-safe.
type Tracker struct {
	// Holds a set of resources(tracking) to set of resources(tracked)
	tracker map[types.NamespacedName]map[types.NamespacedName]struct{}
	mu      sync.RWMutex
}

// NewTracker creates a new Tracker
func NewTracker() *Tracker {
	return &Tracker{tracker: map[types.NamespacedName]map[types.NamespacedName]struct{}{}}
}

// Track records that the tracking object is interested in all tracked objects
func (s *Tracker) Track(tracking types.NamespacedName, tracked ...types.NamespacedName) {
	s.mu.Lock()
	defer s.mu.Unlock()
	set, found := s.tracker[tracking]
	if !found {
		set = map[types.NamespacedName]struct{}{}
		s.tracker[tracking] = set
	}
	for _, t := range tracked {
		set[t] = struct{}{}
	}
}

// UntrackAll untracks all tracking objects. This method is idempotent
func (s *Tracker) UntrackAll(tracking types.NamespacedName) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.tracker, tracking)
}

// GetTracking returns all tracking objects for a given tracked object
func (s *Tracker) GetTracking(tracked types.NamespacedName) []types.NamespacedName {
	s.mu.RLock()
	defer s.mu.RUnlock()
	trackingList := []types.NamespacedName{}
	for tracking, trackedSet := range s.tracker {
		if _, found := trackedSet[tracked]; found {
			trackingList = append(trackingList, tracking)
		}
	}
	return trackingList
}
