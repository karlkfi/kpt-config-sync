package tree_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	v1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/transform/tree"
	"github.com/google/nomos/pkg/policyimporter/analyzer/transform/tree/treetesting"
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
			objects:  []ast.FileObject{fake.HierarchyConfig("system/hc.yaml")},
			expected: treetesting.BuildTree(t, fake.HierarchyConfig("system/hc.yaml")),
		},
		{
			name:     "two objects yields both",
			objects:  []ast.FileObject{fake.HierarchyConfig("system/hc.yaml")},
			expected: treetesting.BuildTree(t, fake.HierarchyConfig("system/hc.yaml")),
		},
		{
			name:    "one object with existing adds object",
			initial: []ast.FileObject{fake.HierarchyConfig("system/hc-1.yaml")},
			objects: []ast.FileObject{fake.HierarchyConfig("system/hc-2.yaml")},
			expected: treetesting.BuildTree(t,
				fake.HierarchyConfig("system/hc-1.yaml"),
				fake.HierarchyConfig("system/hc-2.yaml")),
		},
		{
			name:    "repo yeilds repo",
			objects: []ast.FileObject{fake.Repo("system/repo.yaml")},
			// BuildTree does do this now. This is for illustrative purposes.
			expected: &ast.Root{
				System: &ast.System{
					Objects: []*ast.SystemObject{{FileObject: fake.Repo("system/repo.yaml")}},
				},
				Repo: fake.Repo("system/repo.yaml").Object.(*v1.Repo),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := &ast.Root{}

			if tc.initial != nil {
				actual.System = &ast.System{}
				for _, o := range tc.initial {
					actual.System.Objects = append(actual.System.Objects, &ast.SystemObject{FileObject: o})
				}
			}

			actual.Accept(tree.NewSystemBuilderVisitor(tc.objects))

			if diff := cmp.Diff(tc.expected, actual); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}
