package tree_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/transform/tree"
	"github.com/google/nomos/pkg/importer/analyzer/transform/tree/treetesting"
	"github.com/google/nomos/pkg/testing/fake"
)

func TestClusterBuilderVisitor(t *testing.T) {
	testCases := []struct {
		name     string
		initial  []ast.FileObject
		objects  []ast.FileObject
		expected *ast.Root
	}{
		{
			name:     "empty yields empty Cluster",
			expected: treetesting.BuildTree(t),
		},
		{
			name:     "one object yeilds object",
			objects:  []ast.FileObject{fake.ClusterRoleAtPath("cluster/cr.yaml")},
			expected: treetesting.BuildTree(t, fake.ClusterRoleAtPath("cluster/cr.yaml")),
		},
		{
			name:     "two objects yields both",
			objects:  []ast.FileObject{fake.ClusterRoleAtPath("cluster/cr.yaml")},
			expected: treetesting.BuildTree(t, fake.ClusterRoleAtPath("cluster/cr.yaml")),
		},
		{
			name:    "one object with existing adds object",
			initial: []ast.FileObject{fake.ClusterRoleAtPath("cluster/cr-1.yaml")},
			objects: []ast.FileObject{fake.ClusterRoleAtPath("cluster/cr-2.yaml")},
			expected: treetesting.BuildTree(t,
				fake.ClusterRoleAtPath("cluster/cr-1.yaml"),
				fake.ClusterRoleAtPath("cluster/cr-2.yaml")),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := &ast.Root{}

			if tc.initial != nil {
				for _, o := range tc.initial {
					actual.ClusterObjects = append(actual.ClusterObjects, &ast.ClusterObject{FileObject: o})
				}
			}

			actual.Accept(tree.NewClusterBuilderVisitor(tc.objects))

			if diff := cmp.Diff(tc.expected, actual, cmp.AllowUnexported(ast.FileObject{})); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}
