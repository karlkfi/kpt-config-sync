package parse

import (
	"testing"

	"github.com/google/nomos/pkg/status"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type noOpRemediator struct {
	needsUpdate bool
}

func (r *noOpRemediator) NeedsUpdate() bool {
	return r.needsUpdate
}

func (r *noOpRemediator) UpdateWatches(gvkMap map[schema.GroupVersionKind]struct{}) status.MultiError {
	r.needsUpdate = false
	return nil
}

func TestOpts_StateTracking(t *testing.T) {
	rem := &noOpRemediator{}
	o := &opts{
		updater: updater{
			remediator: rem,
		},
	}
	toApply := "/repo/rev/abcdef"

	// Test initial state (nothing checkpointed yet).
	isUp := o.upToDate(toApply)
	if isUp {
		t.Errorf("got %v from upToDate(); want false", isUp)
	}

	// Checkpoint and verify we are now up-to-date.
	o.checkpoint(toApply)
	isUp = o.upToDate(toApply)
	if !isUp {
		t.Errorf("got %v from upToDate(); want true", isUp)
	}

	// Invalidate and verify we are no longer up-to-date.
	o.invalidate()
	isUp = o.upToDate(toApply)
	if isUp {
		t.Errorf("got %v from upToDate(); want false", isUp)
	}

	// Checkpoint again, but invalidate the remediator. Verify we are not
	// up-to-date.
	o.checkpoint(toApply)
	rem.needsUpdate = true
	isUp = o.upToDate(toApply)
	if isUp {
		t.Errorf("got %v from upToDate(); want false", isUp)
	}

	// Reset the remediator and verify we are up-to-date.
	rem.needsUpdate = false
	isUp = o.upToDate(toApply)
	if !isUp {
		t.Errorf("got %v from upToDate(); want true", isUp)
	}
}
