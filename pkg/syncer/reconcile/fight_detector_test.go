package reconcile

import (
	"testing"
	"time"

	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/testing/fake"
)

// durations creates a sequence of evenly-spaced time.Durations.
// The algorithm is least sensitive to evenly-spaced updates, so triggering
// on such patterns ensures it will trigger on other patterns.
func durations(diff time.Duration, n int) []time.Duration {
	result := make([]time.Duration, n)
	var d time.Duration
	for i := 0; i < n; i++ {
		result[i] = d
		d += diff
	}
	return result
}

var fourUpdatesAtOnce = durations(0, 4)
var sixUpdatesAtOnce = durations(0, 6)

func TestFight(t *testing.T) {
	testCases := []struct {
		name string
		// startHeat is the initial heat at the start of the simulation.
		// This is our initial estimate of updates per minute.
		startHeat float64
		// deltas are a monotonically nondecreasing sequence of update times,
		// as durations measured from the beginning of the simulation.
		deltas []time.Duration
		// whether we want the estimated updates per minute at the end of the
		// test to be above or equal to our threshold.
		wantAboveThreshold bool
	}{
		// Sets of immediate updates.
		{
			name:               "one is below threshold",
			deltas:             durations(0, 1),
			wantAboveThreshold: false,
		},
		{
			name:               "four at once is below threshold",
			deltas:             durations(0, 4),
			wantAboveThreshold: false,
		},
		{
			name:               "six at once is at threshold",
			deltas:             sixUpdatesAtOnce,
			wantAboveThreshold: true,
		},
		// Evenly spread updates takes more time to adjust to.
		{
			name:               "seven over a minute is below threshold",
			deltas:             durations(time.Minute/6.0, 7),
			wantAboveThreshold: false,
		},
		{
			name: "eight over a minute is above threshold",
			// This is sufficient to prove that *any* pattern of eight updates in 1 minute
			// will exceed the heat threshold.
			deltas:             durations(time.Minute/7.0, 8),
			wantAboveThreshold: true,
		},
		// Five updates per minute eventually triggers threshold, but not immediately.
		{
			name:               "five per minute over two minutes is below threshold",
			deltas:             durations(time.Minute/5.0, 10),
			wantAboveThreshold: false,
		},
		{
			name: "five per minute over three minutes is above threshold",
			// As above, this proves that *any* pattern of sixteen updates in 3 minutes
			// will exceed the heat threshold.
			deltas:             durations(time.Minute/5.0, 16),
			wantAboveThreshold: true,
		},
		{
			name:               "starting from high heat does not immediately adjust to lower frequency",
			startHeat:          60.0,
			deltas:             durations(time.Minute/200.0, 4),
			wantAboveThreshold: true,
		},
		{
			name:               "high heat eventually adjusts to new lower frequency",
			startHeat:          60.0,
			deltas:             durations(time.Minute/2.0, 8),
			wantAboveThreshold: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			now := time.Now()
			f := fight{
				heat: tc.startHeat,
				last: now,
			}

			heat := tc.startHeat
			for _, d := range tc.deltas {
				heat = f.markUpdated(now.Add(d))
			}

			if heat < fightThreshold && tc.wantAboveThreshold {
				t.Errorf("got heat = %f < %f, want heat >= %f", heat, fightThreshold, fightThreshold)
			} else if heat >= fightThreshold && !tc.wantAboveThreshold {
				t.Errorf("got heat = %f >= %f, want heat < %f", heat, fightThreshold, fightThreshold)
			}
		})
	}
}

func TestFightDetector(t *testing.T) {
	roleGK := kinds.Role().GroupKind()
	roleBindingGK := kinds.RoleBinding().GroupKind()
	testCases := []struct {
		name               string
		updates            map[gknn][]time.Duration
		wantAboveThreshold map[gknn]bool
	}{
		{
			name: "one object below threshold",
			updates: map[gknn][]time.Duration{
				gknn{gk: roleGK, namespace: "foo", name: "admin"}: fourUpdatesAtOnce,
			},
		},
		{
			name: "one object above threshold",
			updates: map[gknn][]time.Duration{
				gknn{gk: roleGK, namespace: "foo", name: "admin"}: sixUpdatesAtOnce,
			},
			wantAboveThreshold: map[gknn]bool{
				gknn{gk: roleGK, namespace: "foo", name: "admin"}: true,
			},
		},
		{
			name: "four objects objects below threshold",
			updates: map[gknn][]time.Duration{
				gknn{gk: roleGK, namespace: "foo", name: "admin"}:        fourUpdatesAtOnce,
				gknn{gk: roleBindingGK, namespace: "foo", name: "admin"}: fourUpdatesAtOnce,
				gknn{gk: roleGK, namespace: "bar", name: "admin"}:        fourUpdatesAtOnce,
				gknn{gk: roleGK, namespace: "foo", name: "user"}:         fourUpdatesAtOnce,
			},
		},
		{
			name: "two of four objects objects above threshold",
			updates: map[gknn][]time.Duration{
				gknn{gk: roleGK, namespace: "foo", name: "admin"}:        sixUpdatesAtOnce,
				gknn{gk: roleBindingGK, namespace: "foo", name: "admin"}: fourUpdatesAtOnce,
				gknn{gk: roleGK, namespace: "bar", name: "admin"}:        fourUpdatesAtOnce,
				gknn{gk: roleGK, namespace: "foo", name: "user"}:         sixUpdatesAtOnce,
			},
			wantAboveThreshold: map[gknn]bool{
				gknn{gk: roleGK, namespace: "foo", name: "admin"}: true,
				gknn{gk: roleGK, namespace: "foo", name: "user"}:  true,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fd := fightDetector{}

			now := time.Now()
			for o, updates := range tc.updates {
				u := fake.Unstructured(o.gk.WithVersion(""), core.Namespace(o.namespace), core.Name(o.name))

				aboveThreshold := false
				for _, update := range updates {
					nowAbove := fd.markUpdated(now.Add(update), ast.NewFileObject(&u, cmpath.FromSlash("")))
					aboveThreshold = aboveThreshold || nowAbove
				}
				if tc.wantAboveThreshold[o] && !aboveThreshold {
					t.Errorf("got makeUpdated(%v) = false, want true", o)
				} else if !tc.wantAboveThreshold[o] && aboveThreshold {
					t.Errorf("got makeUpdated(%v) = true, want false", o)
				}
			}
		})
	}
}
