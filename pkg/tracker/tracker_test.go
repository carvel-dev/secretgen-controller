// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package tracker_test

import (
	"testing"

	"carvel.dev/secretgen-controller/pkg/tracker"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/types"
)

func Test_Tracker(t *testing.T) {

	t.Run("Test tracker", func(t *testing.T) {
		tracking1 := types.NamespacedName{Namespace: "ns1", Name: "tracking"}
		tracking2 := types.NamespacedName{Namespace: "ns2", Name: "tracking"}
		tracked1 := types.NamespacedName{Namespace: "ns3", Name: "tracked"}
		tracked2 := types.NamespacedName{Namespace: "ns4", Name: "tracked"}
		neverTracked := types.NamespacedName{Namespace: "ns4", Name: "nevertracked"}

		tracker := tracker.NewTracker()

		assert.Len(t, tracker.GetTracking(tracked1), 0, "should be zero tracking")
		assert.Len(t, tracker.GetTracking(tracked2), 0, "should be zero tracking")

		tracker.Track(tracking1, tracked1, tracked2)
		tracker.Track(tracking2, tracked1)

		assert.ElementsMatch(t, tracker.GetTracking(tracked1), []types.NamespacedName{tracking1, tracking2}, "did not contain both tracking resources")
		assert.ElementsMatch(t, tracker.GetTracking(tracked2), []types.NamespacedName{tracking1}, "did not contain tracking resource")

		assert.Len(t, tracker.GetTracking(neverTracked), 0, "should be zero tracking")

		tracker.UntrackAll(tracking1)
		assert.ElementsMatch(t, tracker.GetTracking(tracked1), []types.NamespacedName{tracking2}, "did not contain tracking resource")
		assert.Len(t, tracker.GetTracking(tracked2), 0, "should be zero tracking")

		tracker.UntrackAll(tracking2)
		assert.Len(t, tracker.GetTracking(tracked1), 0, "should be zero tracking")
	})
}
