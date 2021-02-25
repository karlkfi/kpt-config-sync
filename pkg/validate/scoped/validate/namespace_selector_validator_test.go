package validate

import (
	"errors"
	"testing"

	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/testing/fake"
	"github.com/google/nomos/pkg/validate/objects"
)

func TestNamespaceSelectors(t *testing.T) {
	testCases := []struct {
		name     string
		objs     *objects.Scoped
		wantErrs status.MultiError
	}{
		{
			name: "No objects",
			objs: &objects.Scoped{},
		},
		{
			name: "One NamespaceSelector",
			objs: &objects.Scoped{
				Cluster: []ast.FileObject{
					fake.NamespaceSelector(core.Name("first")),
				},
			},
		},
		{
			name: "Two NamespaceSelectors",
			objs: &objects.Scoped{
				Cluster: []ast.FileObject{
					fake.NamespaceSelector(core.Name("first")),
					fake.NamespaceSelector(core.Name("second")),
				},
			},
		},
		{
			name: "Duplicate NamespaceSelectors",
			objs: &objects.Scoped{
				Cluster: []ast.FileObject{
					fake.NamespaceSelector(core.Name("first")),
					fake.NamespaceSelector(core.Name("first")),
				},
			},
			wantErrs: nonhierarchical.SelectorMetadataNameCollisionError(kinds.NamespaceSelector().Kind, "first", fake.NamespaceSelector()),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			errs := NamespaceSelectors(tc.objs)
			if !errors.Is(errs, tc.wantErrs) {
				t.Errorf("got NamespaceSelectors() error %v, want %v", errs, tc.wantErrs)
			}
		})
	}
}
