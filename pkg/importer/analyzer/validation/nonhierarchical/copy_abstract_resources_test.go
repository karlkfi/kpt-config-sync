package nonhierarchical

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/testing/fake"
)

func TestCopyAbstractResources(t *testing.T) {
	testCases := []struct {
		name     string
		before   []ast.FileObject
		expected []ast.FileObject
	}{
		{
			name: "empty makes no changes",
		},
		{
			name:     "keeps cluster-scoped object",
			before:   []ast.FileObject{fake.Cluster()},
			expected: []ast.FileObject{fake.Cluster()},
		},
		{
			name: "keeps namespace-scoped object with Namespace",
			before: []ast.FileObject{
				fake.Namespace("namespaces/foo"),
				fake.Role(core.Namespace("foo")),
			},
			expected: []ast.FileObject{
				fake.Namespace("namespaces/foo"),
				fake.Role(core.Namespace("foo")),
			},
		},
		{
			name: "duplicates namespace-scoped object with namespace-selector",
			before: []ast.FileObject{
				fake.Namespace("namespaces/bar"),
				fake.Namespace("namespaces/foo"),
				fake.Role(core.Annotation(v1.NamespaceSelectorAnnotationKey, "value")),
			},
			expected: []ast.FileObject{
				fake.Namespace("namespaces/bar"),
				fake.Namespace("namespaces/foo"),
				fake.Role(core.Namespace("bar"), core.Annotation(v1.NamespaceSelectorAnnotationKey, "value")),
				fake.Role(core.Namespace("foo"), core.Annotation(v1.NamespaceSelectorAnnotationKey, "value")),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := CopyAbstractResources(tc.before)

			if diff := cmp.Diff(tc.expected, actual); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}
