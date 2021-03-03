package validate

import (
	"errors"
	"testing"

	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/syncer/syncertest"
	"github.com/google/nomos/pkg/testing/fake"
)

func TestUnmanagedNamespaces(t *testing.T) {
	testCases := []struct {
		name     string
		objs     []ast.FileObject
		wantErrs status.MultiError
	}{
		{
			name: "Cluster-scoped objects pass",
			objs: []ast.FileObject{
				fake.ClusterRole(),
				fake.ClusterRole(syncertest.ManagementDisabled),
			},
		},
		{
			name: "Namespace-scoped objects in managed namespace pass",
			objs: []ast.FileObject{
				fake.Namespace("namespaces/foo"),
				fake.Role(core.Namespace("foo")),
				fake.Role(core.Namespace("foo"), syncertest.ManagementDisabled),
			},
		},
		{
			name: "Unmanaged namespace-scoped object in unmanaged namespace passes",
			objs: []ast.FileObject{
				fake.Namespace("namespaces/foo", syncertest.ManagementDisabled),
				fake.Role(core.Namespace("foo"), syncertest.ManagementDisabled),
			},
		},
		{
			name: "Unmanaged namespace-scoped object in managed namespace fails",
			objs: []ast.FileObject{
				fake.Namespace("namespaces/foo", syncertest.ManagementDisabled),
				fake.Role(core.Namespace("foo")),
			},
			wantErrs: fake.Errors(nonhierarchical.ManagedResourceInUnmanagedNamespaceErrorCode),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			errs := UnmanagedNamespaces(tc.objs)
			if !errors.Is(errs, tc.wantErrs) {
				t.Errorf("got UnmanagedNamespaces() error %v, want %v", errs, tc.wantErrs)
			}
		})
	}
}
