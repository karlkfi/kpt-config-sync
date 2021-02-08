package tree_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/transform/tree"
	"github.com/google/nomos/pkg/importer/analyzer/transform/tree/treetesting"
	"github.com/google/nomos/pkg/testing/fake"
)

func TestSystemBuilderVisitor(t *testing.T) {
	testCases := []struct {
		name     string
		initial  []ast.FileObject
		objects  []ast.FileObject
		expected *ast.Root
	}{
		{
			name:     "empty yields empty System",
			expected: treetesting.BuildTree(t),
		},
		{
			name:     "one object yeilds object",
			objects:  []ast.FileObject{fake.HierarchyConfigAtPath("system/hc.yaml")},
			expected: treetesting.BuildTree(t, fake.HierarchyConfigAtPath("system/hc.yaml")),
		},
		{
			name:     "two objects yields both",
			objects:  []ast.FileObject{fake.HierarchyConfigAtPath("system/hc.yaml")},
			expected: treetesting.BuildTree(t, fake.HierarchyConfigAtPath("system/hc.yaml")),
		},
		{
			name:    "one object with existing adds object",
			initial: []ast.FileObject{fake.HierarchyConfigAtPath("system/hc-1.yaml")},
			objects: []ast.FileObject{fake.HierarchyConfigAtPath("system/hc-2.yaml")},
			expected: treetesting.BuildTree(t,
				fake.HierarchyConfigAtPath("system/hc-1.yaml"),
				fake.HierarchyConfigAtPath("system/hc-2.yaml")),
		},
		{
			name:    "repo yeilds repo",
			objects: []ast.FileObject{fake.Repo()},
			// BuildTree does do this now. This is for illustrative purposes.
			expected: &ast.Root{
				SystemObjects: []*ast.SystemObject{{FileObject: fake.Repo()}},
				Repo:          fake.RepoObject(),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := &ast.Root{}

			if tc.initial != nil {
				for _, o := range tc.initial {

					actual.SystemObjects = append(actual.SystemObjects, &ast.SystemObject{FileObject: o})
				}
			}

			actual.Accept(tree.NewSystemBuilderVisitor(tc.objects))

			if diff := cmp.Diff(tc.expected, actual, ast.CompareFileObject); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}
