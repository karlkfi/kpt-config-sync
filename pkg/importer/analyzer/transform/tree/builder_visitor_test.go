package tree_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/ast/node"
	"github.com/google/nomos/pkg/importer/analyzer/transform/tree"
	"github.com/google/nomos/pkg/importer/analyzer/transform/tree/treetesting"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/testing/fake"
)

func TestBuilderVisitor(t *testing.T) {
	testCases := []struct {
		name    string
		objects []ast.FileObject
		// expected is the manual long form version of the entire config hierarchy that Builder is
		// expected to produce.
		expected *ast.Root
		// expectedEquivalent is the short form made possible by treetesting.BuildTree
		// These tests verify that the two forms are equivalent.
		expectedEquivalent *ast.Root
	}{
		{
			name:               "no objects",
			expected:           &ast.Root{},
			expectedEquivalent: treetesting.BuildTree(t),
		},
		{
			name: "in leaf directories ",
			objects: []ast.FileObject{
				fake.RoleAtPath("namespaces/foo/bar/role.yaml"),
				fake.RoleAtPath("namespaces/qux/role.yaml"),
			},
			expected: &ast.Root{
				Tree: &ast.TreeNode{
					Relative: cmpath.RelativeSlash("namespaces"),
					Type:     node.AbstractNamespace,
					Children: []*ast.TreeNode{
						{
							Relative: cmpath.RelativeSlash("namespaces/foo"),
							Type:     node.AbstractNamespace,
							Children: []*ast.TreeNode{
								{
									Relative: cmpath.RelativeSlash("namespaces/foo/bar"),
									Type:     node.AbstractNamespace,
									Objects:  []*ast.NamespaceObject{{FileObject: fake.RoleAtPath("namespaces/foo/bar/role.yaml")}},
								},
							},
						},
						{
							Relative: cmpath.RelativeSlash("namespaces/qux"),
							Type:     node.AbstractNamespace,
							Objects:  []*ast.NamespaceObject{{FileObject: fake.RoleAtPath("namespaces/qux/role.yaml")}},
						},
					},
				},
			},
			expectedEquivalent: treetesting.BuildTree(t, fake.RoleAtPath("namespaces/qux/role.yaml"), fake.RoleAtPath("namespaces/foo/bar/role.yaml")),
		},
		{
			name: "two in same directory",
			objects: []ast.FileObject{
				fake.RoleAtPath("namespaces/foo/bar/role.yaml"),
				fake.RoleAtPath("namespaces/qux/role.yaml"),
				fake.RoleBindingAtPath("namespaces/qux/rolebinding.yaml"),
			},
			expected: &ast.Root{
				Tree: &ast.TreeNode{
					Relative: cmpath.RelativeSlash("namespaces"),
					Type:     node.AbstractNamespace,
					Children: []*ast.TreeNode{
						{
							Relative: cmpath.RelativeSlash("namespaces/foo"),
							Type:     node.AbstractNamespace,
							Children: []*ast.TreeNode{
								{
									Relative: cmpath.RelativeSlash("namespaces/foo/bar"),
									Type:     node.AbstractNamespace,
									Objects:  []*ast.NamespaceObject{{FileObject: fake.RoleAtPath("namespaces/foo/bar/role.yaml")}},
								},
							},
						},
						{
							Relative: cmpath.RelativeSlash("namespaces/qux"),
							Type:     node.AbstractNamespace,
							Objects: []*ast.NamespaceObject{
								{FileObject: fake.RoleAtPath("namespaces/qux/role.yaml")},
								{FileObject: fake.RoleBindingAtPath("namespaces/qux/rolebinding.yaml")},
							},
						},
					},
				},
			},
			expectedEquivalent: treetesting.BuildTree(t,
				fake.RoleAtPath("namespaces/foo/bar/role.yaml"),
				fake.RoleAtPath("namespaces/qux/role.yaml"),
				fake.RoleBindingAtPath("namespaces/qux/rolebinding.yaml")),
		},
		{
			name: "in non-leaf child of hierarchy root",
			objects: []ast.FileObject{
				fake.RoleAtPath("namespaces/foo/bar/role.yaml"),
				fake.RoleAtPath("namespaces/foo/role.yaml"),
				fake.RoleBindingAtPath("namespaces/qux/rolebinding.yaml"),
			},
			expected: &ast.Root{
				Tree: &ast.TreeNode{
					Relative: cmpath.RelativeSlash("namespaces"),
					Type:     node.AbstractNamespace,
					Children: []*ast.TreeNode{
						{
							Relative: cmpath.RelativeSlash("namespaces/foo"),
							Type:     node.AbstractNamespace,
							Objects:  []*ast.NamespaceObject{{FileObject: fake.RoleAtPath("namespaces/foo/role.yaml")}},
							Children: []*ast.TreeNode{
								{
									Relative: cmpath.RelativeSlash("namespaces/foo/bar"),
									Type:     node.AbstractNamespace,
									Objects:  []*ast.NamespaceObject{{FileObject: fake.RoleAtPath("namespaces/foo/bar/role.yaml")}},
								},
							},
						},
						{
							Relative: cmpath.RelativeSlash("namespaces/qux"),
							Type:     node.AbstractNamespace,
							Objects:  []*ast.NamespaceObject{{FileObject: fake.RoleBindingAtPath("namespaces/qux/rolebinding.yaml")}},
						},
					},
				},
			},
			expectedEquivalent: treetesting.BuildTree(t,
				fake.RoleAtPath("namespaces/foo/bar/role.yaml"),
				fake.RoleAtPath("namespaces/foo/role.yaml"),
				fake.RoleBindingAtPath("namespaces/qux/rolebinding.yaml")),
		},
		{
			name: "namespace in leaf node",
			objects: []ast.FileObject{
				fake.Namespace("namespaces/foo/bar"),
				fake.Namespace("namespaces/qux"),
			},
			expected: &ast.Root{
				Tree: &ast.TreeNode{
					Relative: cmpath.RelativeSlash("namespaces"),
					Type:     node.AbstractNamespace,
					Children: []*ast.TreeNode{
						{
							Relative: cmpath.RelativeSlash("namespaces/foo"),
							Type:     node.AbstractNamespace,
							Children: []*ast.TreeNode{
								{
									Relative: cmpath.RelativeSlash("namespaces/foo/bar"),
									Type:     node.Namespace,
									Objects:  []*ast.NamespaceObject{{FileObject: fake.Namespace("namespaces/foo/bar")}},
								},
							},
						},
						{
							Relative: cmpath.RelativeSlash("namespaces/qux"),
							Type:     node.Namespace,
							Objects:  []*ast.NamespaceObject{{FileObject: fake.Namespace("namespaces/qux")}},
						},
					},
				},
			},
			expectedEquivalent: treetesting.BuildTree(t,
				fake.Namespace("namespaces/foo/bar"),
				fake.Namespace("namespaces/qux")),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			v := tree.NewBuilderVisitor(tc.objects)

			actual := &ast.Root{}
			actual.Accept(v)

			if diff := cmp.Diff(tc.expected, actual, cmp.AllowUnexported(ast.FileObject{}), cmpopts.EquateEmpty()); diff != "" {
				t.Fatalf("unexpected difference in trees\n\n%s", diff)
			}

			if diff := cmp.Diff(tc.expectedEquivalent, actual, cmp.AllowUnexported(ast.FileObject{})); diff != "" {
				t.Fatalf("unexpected non-equivalent trees\n\n%s", diff)
			}
		})
	}
}
