package transform

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/ast/node"
	"github.com/google/nomos/pkg/importer/analyzer/transform/tree/treetesting"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
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
			objects: []ast.FileObject{fake.Namespace("namespaces/bar")},
			expected: &ast.Root{
				Tree: &ast.TreeNode{
					Path: cmpath.FromSlash("namespaces"),
					Type: node.AbstractNamespace,
					Children: []*ast.TreeNode{
						{
							Path: cmpath.FromSlash("namespaces/bar"),
							Type: node.Namespace,
						},
					},
				},
			},
		},
		{
			name:    "namespaceselector returns empty",
			objects: []ast.FileObject{fake.NamespaceSelector(core.Name(""))},
			expected: &ast.Root{
				Tree: &ast.TreeNode{
					Path: cmpath.FromSlash("namespaces"),
					Type: node.AbstractNamespace,
					Selectors: map[string]*v1.NamespaceSelector{
						"": fake.NamespaceSelectorObject(core.Name("")),
					},
				},
			},
		},
		{
			name:     "keeps non-ephemeral",
			objects:  []ast.FileObject{fake.RoleAtPath("namespaces/bar/role.yaml")},
			expected: treetesting.BuildTree(t, fake.RoleAtPath("namespaces/bar/role.yaml")),
		},
		{
			name:    "only non-ephemeral",
			objects: []ast.FileObject{fake.Namespace("namespaces/bar"), fake.RoleAtPath("namespaces/bar/role.yaml")},
			expected: &ast.Root{
				Tree: &ast.TreeNode{
					Path: cmpath.FromSlash("namespaces"),
					Type: node.AbstractNamespace,
					Children: []*ast.TreeNode{
						{
							Path:    cmpath.FromSlash("namespaces/bar"),
							Type:    node.Namespace,
							Objects: []*ast.NamespaceObject{{FileObject: fake.RoleAtPath("namespaces/bar/role.yaml")}},
						},
					},
				},
			},
		},
		{
			name:    "role in System returns same",
			objects: []ast.FileObject{fake.RoleAtPath("system/role.yaml")},
			expected: &ast.Root{
				SystemObjects: []*ast.SystemObject{{FileObject: fake.RoleAtPath("system/role.yaml")}},
			},
		},
		{
			name:    "role in ClusterRegistry returns same",
			objects: []ast.FileObject{fake.RoleAtPath("clusterregistry/role.yaml")},
			expected: &ast.Root{
				ClusterRegistryObjects: []*ast.ClusterRegistryObject{{FileObject: fake.RoleAtPath("clusterregistry/role.yaml")}},
			},
		},
		{
			name:    "role in Cluster returns same",
			objects: []ast.FileObject{fake.RoleAtPath("cluster")},
			expected: &ast.Root{
				ClusterObjects: []*ast.ClusterObject{{FileObject: fake.RoleAtPath("cluster")}},
			},
		},
		{
			name:     "namespace in System returns empty",
			objects:  []ast.FileObject{fake.Namespace("system")},
			expected: &ast.Root{},
		},
		{
			name:     "namespace in ClusterRegistry returns empty",
			objects:  []ast.FileObject{fake.Namespace("clusterregistry")},
			expected: &ast.Root{},
		},
		{
			name:     "namespace in Cluster returns empty",
			objects:  []ast.FileObject{fake.Namespace("cluster")},
			expected: &ast.Root{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			root := treetesting.BuildTree(t, tc.objects...)

			root.Accept(NewEphemeralResourceRemover())

			if diff := cmp.Diff(tc.expected, root, cmp.AllowUnexported(ast.FileObject{}), cmpopts.EquateEmpty()); diff != "" {
				t.Fatalf("unexpected difference in trees\n\n%s", diff)
			}
		})
	}
}
