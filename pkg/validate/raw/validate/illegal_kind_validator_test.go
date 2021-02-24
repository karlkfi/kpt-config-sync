package validate

import (
	"errors"
	"testing"

	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/testing/fake"
)

func TestIllegalKindsForUnstructured(t *testing.T) {
	testCases := []struct {
		name    string
		obj     ast.FileObject
		wantErr status.Error
	}{
		{
			name: "Non-hierarchical object passes",
			obj:  fake.ClusterSelector(),
		},
		{
			name:    "HierarchyConfig object fails",
			obj:     fake.HierarchyConfig(),
			wantErr: nonhierarchical.IllegalHierarchicalKind(fake.HierarchyConfig()),
		},
		{
			name:    "Repo object fails",
			obj:     fake.Repo(),
			wantErr: nonhierarchical.IllegalHierarchicalKind(fake.Repo()),
		},
		{
			name:    "Sync object fails",
			obj:     fake.FileObject(fake.SyncObject(kinds.Role().GroupKind()), "sync.yaml"),
			wantErr: nonhierarchical.UnsupportedObjectError(fake.SyncObject(kinds.Role().GroupKind())),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := IllegalKindsForUnstructured(tc.obj)
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("got IllegalKindsForUnstructured() error %v, want %v", err, tc.wantErr)
			}
		})
	}
}
