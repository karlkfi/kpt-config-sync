package fake

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// StatusWriterRecorder records the runtime.Objects passed to Update().
type StatusWriterRecorder struct {
	updates []runtime.Object
}

// Update implements client.StatusWriter.
func (s *StatusWriterRecorder) Update(ctx context.Context, obj runtime.Object, opts ...client.UpdateOption) error {
	if len(opts) > 0 {
		jsn, _ := json.MarshalIndent(opts, "", "  ")
		return errors.Errorf("fake.StatusWriter.Update does not yet support opts, but got: %v", string(jsn))
	}

	s.updates = append(s.updates, obj)
	return nil
}

// Patch implements client.StatusWriter.
func (s *StatusWriterRecorder) Patch(context.Context, runtime.Object, client.Patch, ...client.PatchOption) error {
	return errors.New("fake.StatusWriter does not support Patch()")
}

// Check ensures the StatusWriterRecorder got the correct set of updates to Syncs.
func (s *StatusWriterRecorder) Check(t *testing.T, wantUpdates ...runtime.Object) {
	t.Helper()
	if diff := cmp.Diff(wantUpdates, s.updates, cmpopts.EquateEmpty()); diff != "" {
		t.Error("diff to StatusWriter.Update() calls", diff)
	}
}

var _ client.StatusWriter = &StatusWriterRecorder{}
