package parse

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/declared"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/testing/fake"
	"github.com/pkg/errors"
)

func TestNamespaceScopeVisitor(t *testing.T) {
	testCases := []struct {
		name    string
		scope   declared.Scope
		obj     ast.FileObject
		want    ast.FileObject
		wantErr status.Error
	}{
		{
			name:  "correct Namespace pass",
			scope: "foo",
			obj:   fake.Role(core.Namespace("foo")),
		},
		{
			name:  "blank Namespace pass and update Namespace",
			scope: "foo",
			obj:   fake.Role(core.Namespace("")),
			want:  fake.Role(core.Namespace("foo")),
		},
		{
			name:    "wrong Namespace error",
			scope:   "foo",
			obj:     fake.Role(core.Namespace("bar")),
			wantErr: nonhierarchical.BadScopeErrBuilder.Sprint("").BuildWithResources(fake.Role()),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.want.Unstructured == nil {
				// We don't expect repositoryScopeVisitor to mutate the object.
				tc.want = tc.obj.DeepCopy()
			}

			visitor := repositoryScopeVisitor(tc.scope)

			_, err := visitor([]ast.FileObject{tc.obj})
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("got error %v, want %v", err, tc.wantErr)
			}

			if diff := cmp.Diff(tc.want, tc.obj, ast.CompareFileObject); diff != "" {
				// Either the visitor didn't mutate the object, or it unexpectedly did so.
				t.Error(diff)
			}
		})
	}
}
