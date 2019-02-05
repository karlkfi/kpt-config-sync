package tree_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast/asttesting"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast/node"
	"github.com/google/nomos/pkg/policyimporter/analyzer/transform/tree"
	"github.com/google/nomos/pkg/policyimporter/analyzer/transform/tree/treetesting"
	"github.com/google/nomos/pkg/policyimporter/filesystem/nomospath"
	corev1 "k8s.io/api/core/v1"
)

func fakeRole(path string) ast.FileObject {
	return asttesting.NewFakeFileObject(kinds.Role(), path)
}

func fakeRoleBinding(path string) ast.FileObject {
	return asttesting.NewFakeFileObject(kinds.RoleBinding(), path)
}

func namespace(path string) ast.FileObject {
	return ast.FileObject{
		Relative: nomospath.NewFakeRelative(path),
		Object:   &corev1.Namespace{},
	}
}

func TestBuilderVisitor(t *testing.T) {
	testCases := []struct {
		name    string
		objects []ast.FileObject
		// expected is the manual long form version of the entire policy hierarchy that Builder is
		// expected to produce.
		expected *ast.Root
		// expectedEquivalent is the short form made possible by treetesting.BuildTree
		// These tests verify that the two forms are equivalent.
		expectedEquivalent *ast.Root
	}{
		{
			name:               "no objects",
			expected:           &ast.Root{},
			expectedEquivalent: treetesting.BuildTree(),
		},
		{
			name: "in leaf directories ",
			objects: []ast.FileObject{
				fakeRole("namespaces/foo/bar/role.yaml"),
				fakeRole("namespaces/qux/role.yaml"),
			},
			expected: &ast.Root{
				Tree: &ast.TreeNode{
					Relative: nomospath.NewFakeRelative("namespaces"),
					Type:     node.AbstractNamespace,
					Children: []*ast.TreeNode{
						{
							Relative: nomospath.NewFakeRelative("namespaces/foo"),
							Type:     node.AbstractNamespace,
							Children: []*ast.TreeNode{
								{
									Relative: nomospath.NewFakeRelative("namespaces/foo/bar"),
									Type:     node.AbstractNamespace,
									Objects:  []*ast.NamespaceObject{{FileObject: fakeRole("namespaces/foo/bar/role.yaml")}},
								},
							},
						},
						{
							Relative: nomospath.NewFakeRelative("namespaces/qux"),
							Type:     node.AbstractNamespace,
							Objects:  []*ast.NamespaceObject{{FileObject: fakeRole("namespaces/qux/role.yaml")}},
						},
					},
				},
			},
			expectedEquivalent: treetesting.BuildTree(fakeRole("namespaces/qux/role.yaml"), fakeRole("namespaces/foo/bar/role.yaml")),
		},
		{
			name: "two in same directory",
			objects: []ast.FileObject{
				fakeRole("namespaces/foo/bar/role.yaml"),
				fakeRole("namespaces/qux/role.yaml"),
				fakeRoleBinding("namespaces/qux/rolebinding.yaml"),
			},
			expected: &ast.Root{
				Tree: &ast.TreeNode{
					Relative: nomospath.NewFakeRelative("namespaces"),
					Type:     node.AbstractNamespace,
					Children: []*ast.TreeNode{
						{
							Relative: nomospath.NewFakeRelative("namespaces/foo"),
							Type:     node.AbstractNamespace,
							Children: []*ast.TreeNode{
								{
									Relative: nomospath.NewFakeRelative("namespaces/foo/bar"),
									Type:     node.AbstractNamespace,
									Objects:  []*ast.NamespaceObject{{FileObject: fakeRole("namespaces/foo/bar/role.yaml")}},
								},
							},
						},
						{
							Relative: nomospath.NewFakeRelative("namespaces/qux"),
							Type:     node.AbstractNamespace,
							Objects: []*ast.NamespaceObject{
								{FileObject: fakeRole("namespaces/qux/role.yaml")},
								{FileObject: fakeRoleBinding("namespaces/qux/rolebinding.yaml")},
							},
						},
					},
				},
			},
			expectedEquivalent: treetesting.BuildTree(
				fakeRole("namespaces/foo/bar/role.yaml"),
				fakeRole("namespaces/qux/role.yaml"),
				fakeRoleBinding("namespaces/qux/rolebinding.yaml")),
		},
		{
			name: "in non-leaf child of hierarchy root",
			objects: []ast.FileObject{
				fakeRole("namespaces/foo/bar/role.yaml"),
				fakeRole("namespaces/foo/role.yaml"),
				fakeRoleBinding("namespaces/qux/rolebinding.yaml"),
			},
			expected: &ast.Root{
				Tree: &ast.TreeNode{
					Relative: nomospath.NewFakeRelative("namespaces"),
					Type:     node.AbstractNamespace,
					Children: []*ast.TreeNode{
						{
							Relative: nomospath.NewFakeRelative("namespaces/foo"),
							Type:     node.AbstractNamespace,
							Objects:  []*ast.NamespaceObject{{FileObject: fakeRole("namespaces/foo/role.yaml")}},
							Children: []*ast.TreeNode{
								{
									Relative: nomospath.NewFakeRelative("namespaces/foo/bar"),
									Type:     node.AbstractNamespace,
									Objects:  []*ast.NamespaceObject{{FileObject: fakeRole("namespaces/foo/bar/role.yaml")}},
								},
							},
						},
						{
							Relative: nomospath.NewFakeRelative("namespaces/qux"),
							Type:     node.AbstractNamespace,
							Objects:  []*ast.NamespaceObject{{FileObject: fakeRoleBinding("namespaces/qux/rolebinding.yaml")}},
						},
					},
				},
			},
			expectedEquivalent: treetesting.BuildTree(
				fakeRole("namespaces/foo/bar/role.yaml"),
				fakeRole("namespaces/foo/role.yaml"),
				fakeRoleBinding("namespaces/qux/rolebinding.yaml")),
		},
		{
			name: "namespace in leaf node",
			objects: []ast.FileObject{
				namespace("namespaces/foo/bar/namespace.yaml"),
				namespace("namespaces/qux/namespace.yaml"),
			},
			expected: &ast.Root{
				Tree: &ast.TreeNode{
					Relative: nomospath.NewFakeRelative("namespaces"),
					Type:     node.AbstractNamespace,
					Children: []*ast.TreeNode{
						{
							Relative: nomospath.NewFakeRelative("namespaces/foo"),
							Type:     node.AbstractNamespace,
							Children: []*ast.TreeNode{
								{
									Relative: nomospath.NewFakeRelative("namespaces/foo/bar"),
									Type:     node.Namespace,
									Objects:  []*ast.NamespaceObject{{FileObject: namespace("namespaces/foo/bar/namespace.yaml")}},
								},
							},
						},
						{
							Relative: nomospath.NewFakeRelative("namespaces/qux"),
							Type:     node.Namespace,
							Objects:  []*ast.NamespaceObject{{FileObject: namespace("namespaces/qux/namespace.yaml")}},
						},
					},
				},
			},
			expectedEquivalent: treetesting.BuildTree(
				namespace("namespaces/foo/bar/namespace.yaml"),
				namespace("namespaces/qux/namespace.yaml")),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			v := tree.NewBuilderVisitor(tc.objects)

			actual := &ast.Root{}
			actual.Accept(v)

			if diff := cmp.Diff(tc.expected, actual); diff != "" {
				t.Fatalf("unexpected difference in trees\n\n%s", diff)
			}

			if diff := cmp.Diff(tc.expectedEquivalent, actual); diff != "" {
				t.Fatalf("unexpected non-equivalent trees\n\n%s", diff)
			}
		})
	}
}
