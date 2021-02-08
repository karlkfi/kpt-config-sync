package tree_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/transform/tree"
	"github.com/google/nomos/pkg/importer/analyzer/transform/tree/treetesting"
	"github.com/google/nomos/pkg/testing/fake"
)

func TestClusterRegistryBuilderVisitor(t *testing.T) {
	testCases := []struct {
		name     string
		initial  []ast.FileObject
		objects  []ast.FileObject
		expected *ast.Root
	}{
		{
			name:     "empty yields empty ClusterRegistry",
			expected: treetesting.BuildTree(t),
		},
		{
			name:     "one object yeilds object",
			objects:  []ast.FileObject{fake.ClusterAtPath("clusterregistry/cr.yaml")},
			expected: treetesting.BuildTree(t, fake.ClusterAtPath("clusterregistry/cr.yaml")),
		},
		{
			name:     "two objects yields both",
			objects:  []ast.FileObject{fake.ClusterAtPath("clusterregistry/cr.yaml")},
			expected: treetesting.BuildTree(t, fake.ClusterAtPath("clusterregistry/cr.yaml")),
		},
		{
			name:    "one object with existing adds object",
			initial: []ast.FileObject{fake.ClusterAtPath("clusterregistry/cr-1.yaml")},
			objects: []ast.FileObject{fake.ClusterAtPath("clusterregistry/cr-2.yaml")},
			expected: treetesting.BuildTree(t,
				fake.ClusterAtPath("clusterregistry/cr-1.yaml"),
				fake.ClusterAtPath("clusterregistry/cr-2.yaml")),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := &ast.Root{}

			if tc.initial != nil {
				for _, o := range tc.initial {
					actual.ClusterRegistryObjects = append(actual.ClusterRegistryObjects, &ast.ClusterRegistryObject{FileObject: o})
				}
			}

			actual.Accept(tree.NewClusterRegistryBuilderVisitor(tc.objects))

			if diff := cmp.Diff(tc.expected, actual, ast.CompareFileObject); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}
