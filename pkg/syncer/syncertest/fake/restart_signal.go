package fake

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

// RestartSignalRecorder implements a fake sync.RestartSignal.
type RestartSignalRecorder struct {
	Restarts []string
}

// Restart implements RestartSignal.
func (r *RestartSignalRecorder) Restart(signal string) {
	r.Restarts = append(r.Restarts, signal)
}

// Check ensures that the RestartSignal was called exactly with the passed
// sequence of signals.
func (r *RestartSignalRecorder) Check(t *testing.T, want ...string) {
	if diff := cmp.Diff(want, r.Restarts); diff != "" {
		t.Errorf("Diff in calls to fake.RestartSignalRecorder.Restart(): %s", diff)
	}
}
