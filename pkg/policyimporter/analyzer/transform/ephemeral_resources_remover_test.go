package transform

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	v1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast/node"
	"github.com/google/nomos/pkg/policyimporter/analyzer/transform/tree/treetesting"
	"github.com/google/nomos/pkg/policyimporter/filesystem/nomospath"
	"github.com/google/nomos/pkg/testing/fake"
)

func TestEphemeralResourceRemover(t *testing.T) {
	testCases := []struct {
		name     string
		objects  []ast.FileObject
		expected *ast.Root
	}{
		{
			name:     "empty returns empty",
			expected: &ast.Root{},
		},
		{
			name:    "namespace returns empty",
			objects: []ast.FileObject{fake.Namespace("namespaces/bar/ns.yaml")},
			expected: &ast.Root{
				Tree: &ast.TreeNode{
					Relative: nomospath.NewRelative("namespaces"),
					Type:     node.AbstractNamespace,
					Children: []*ast.TreeNode{
						{
							Relative: nomospath.NewRelative("namespaces/bar"),
							Type:     node.Namespace,
						},
					},
				},
			},
		},
		{
			name:    "namespaceselector returns empty",
			objects: []ast.FileObject{fake.NamespaceSelector("namespaces/ns.yaml")},
			expected: &ast.Root{
				Tree: &ast.TreeNode{
					Relative: nomospath.NewRelative("namespaces"),
					Type:     node.AbstractNamespace,
					Selectors: map[string]*v1.NamespaceSelector{
						"": fake.NamespaceSelector("namespaces/ns.yaml").Object.(*v1.NamespaceSelector),
					},
				},
			},
		},
		{
			name:     "keeps non-ephemeral",
			objects:  []ast.FileObject{fake.Role("namespaces/bar/role.yaml")},
			expected: treetesting.BuildTree(t, fake.Role("namespaces/bar/role.yaml")),
		},
		{
			name:    "only non-ephemeral",
			objects: []ast.FileObject{fake.Namespace("namespaces/bar/ns.yaml"), fake.Role("namespaces/bar/role.yaml")},
			expected: &ast.Root{
				Tree: &ast.TreeNode{
					Relative: nomospath.NewRelative("namespaces"),
					Type:     node.AbstractNamespace,
					Children: []*ast.TreeNode{
						{
							Relative: nomospath.NewRelative("namespaces/bar"),
							Type:     node.Namespace,
							Objects:  []*ast.NamespaceObject{{FileObject: fake.Role("namespaces/bar/role.yaml")}},
						},
					},
				},
			},
		},
		{
			name:    "role in System returns same",
			objects: []ast.FileObject{fake.Role("system/role.yaml")},
			expected: &ast.Root{
				SystemObjects: []*ast.SystemObject{{FileObject: fake.Role("system/role.yaml")}},
			},
		},
		{
			name:    "role in ClusterRegistry returns same",
			objects: []ast.FileObject{fake.Role("clusterregistry/role.yaml")},
			expected: &ast.Root{
				ClusterRegistryObjects: []*ast.ClusterRegistryObject{{FileObject: fake.Role("clusterregistry/role.yaml")}},
			},
		},
		{
			name:    "role in Cluster returns same",
			objects: []ast.FileObject{fake.Role("cluster/role.yaml")},
			expected: &ast.Root{
				ClusterObjects: []*ast.ClusterObject{{FileObject: fake.Role("cluster/role.yaml")}},
			},
		},
		{
			name:     "namespace in System returns empty",
			objects:  []ast.FileObject{fake.Namespace("system/ns.yaml")},
			expected: &ast.Root{},
		},
		{
			name:     "namespace in ClusterRegistry returns empty",
			objects:  []ast.FileObject{fake.Namespace("clusterregistry/ns.yaml")},
			expected: &ast.Root{},
		},
		{
			name:     "namespace in Cluster returns empty",
			objects:  []ast.FileObject{fake.Namespace("cluster/ns.yaml")},
			expected: &ast.Root{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			root := treetesting.BuildTree(t, tc.objects...)

			root.Accept(NewEphemeralResourceRemover())

			if diff := cmp.Diff(tc.expected, root); diff != "" {
				t.Fatalf("unexpected difference in trees\n\n%s", diff)
			}
		})
	}
}
