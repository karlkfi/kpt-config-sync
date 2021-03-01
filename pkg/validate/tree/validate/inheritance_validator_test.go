package validate

import (
	"errors"
	"testing"

	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/ast/node"
	"github.com/google/nomos/pkg/importer/analyzer/validation/semantic"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/testing/fake"
	"github.com/google/nomos/pkg/validate/objects"
)

func TestInheritance(t *testing.T) {
	testCases := []struct {
		name     string
		objs     *objects.Tree
		wantErrs status.MultiError
	}{
		{
			name: "empty tree",
			objs: &objects.Tree{
				Tree: &ast.TreeNode{
					Relative: cmpath.RelativeSlash("namespaces"),
					Type:     node.AbstractNamespace,
				},
			},
		},
		{
			name: "Namespace without resources",
			objs: &objects.Tree{
				Tree: &ast.TreeNode{
					Relative: cmpath.RelativeSlash("namespaces"),
					Type:     node.AbstractNamespace,
					Children: []*ast.TreeNode{
						{
							Relative: cmpath.RelativeSlash("namespaces/hello"),
							Type:     node.Namespace,
							Objects: []*ast.NamespaceObject{
								{FileObject: fake.Namespace("namespaces/hello")},
							},
						},
					},
				},
			},
		},
		{
			name: "Namespace with resource",
			objs: &objects.Tree{
				Tree: &ast.TreeNode{
					Relative: cmpath.RelativeSlash("namespaces"),
					Type:     node.AbstractNamespace,
					Children: []*ast.TreeNode{
						{
							Relative: cmpath.RelativeSlash("namespaces/hello"),
							Type:     node.Namespace,
							Objects: []*ast.NamespaceObject{
								{FileObject: fake.Namespace("namespaces/hello")},
								{FileObject: fake.RoleAtPath("namespaces/hello")},
							},
						},
					},
				},
			},
		},
		{
			name: "abstract namespace with ephemeral resource",
			objs: &objects.Tree{
				Tree: &ast.TreeNode{
					Relative: cmpath.RelativeSlash("namespaces"),
					Type:     node.AbstractNamespace,
					Children: []*ast.TreeNode{
						{
							Relative: cmpath.RelativeSlash("namespaces/hello"),
							Type:     node.AbstractNamespace,
							Objects: []*ast.NamespaceObject{
								{FileObject: fake.NamespaceSelector()},
							},
						},
					},
				},
			},
		},
		{
			name: "abstract namespace with resource and child namespace",
			objs: &objects.Tree{
				Tree: &ast.TreeNode{
					Relative: cmpath.RelativeSlash("namespaces"),
					Type:     node.AbstractNamespace,
					Children: []*ast.TreeNode{
						{
							Relative: cmpath.RelativeSlash("namespaces/hello"),
							Type:     node.AbstractNamespace,
							Objects: []*ast.NamespaceObject{
								{FileObject: fake.RoleAtPath("namespaces/hello")},
							},
							Children: []*ast.TreeNode{
								{
									Relative: cmpath.RelativeSlash("namespaces/hello/world"),
									Type:     node.Namespace,
									Objects: []*ast.NamespaceObject{
										{FileObject: fake.Namespace("namespaces/hello/world")},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "abstract namespace with resource and descendant namespace",
			objs: &objects.Tree{
				Tree: &ast.TreeNode{
					Relative: cmpath.RelativeSlash("namespaces"),
					Type:     node.AbstractNamespace,
					Children: []*ast.TreeNode{
						{
							Relative: cmpath.RelativeSlash("namespaces/hello"),
							Type:     node.AbstractNamespace,
							Objects: []*ast.NamespaceObject{
								{FileObject: fake.RoleAtPath("namespaces/hello")},
							},
							Children: []*ast.TreeNode{
								{
									Relative: cmpath.RelativeSlash("namespaces/hello/world"),
									Type:     node.AbstractNamespace,
									Children: []*ast.TreeNode{
										{
											Relative: cmpath.RelativeSlash("namespaces/hello/world/end"),
											Type:     node.Namespace,
											Objects: []*ast.NamespaceObject{
												{FileObject: fake.Namespace("namespaces/hello/world/end")},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "abstract namespace with resource and no namespace child",
			objs: &objects.Tree{
				Tree: &ast.TreeNode{
					Relative: cmpath.RelativeSlash("namespaces"),
					Type:     node.AbstractNamespace,
					Children: []*ast.TreeNode{
						{
							Relative: cmpath.RelativeSlash("namespaces/hello"),
							Type:     node.AbstractNamespace,
							Objects: []*ast.NamespaceObject{
								{FileObject: fake.RoleAtPath("namespaces/hello")},
							},
						},
					},
				},
			},
			wantErrs: semantic.UnsyncableResourcesInLeaf(&ast.TreeNode{}),
		},
		{
			name: "abstract namespace with resource and abstract child",
			objs: &objects.Tree{
				Tree: &ast.TreeNode{
					Relative: cmpath.RelativeSlash("namespaces"),
					Type:     node.AbstractNamespace,
					Children: []*ast.TreeNode{
						{
							Relative: cmpath.RelativeSlash("namespaces/hello"),
							Type:     node.AbstractNamespace,
							Objects: []*ast.NamespaceObject{
								{FileObject: fake.RoleAtPath("namespaces/hello")},
							},
							Children: []*ast.TreeNode{
								{
									Relative: cmpath.RelativeSlash("namespaces/hello/world"),
									Type:     node.AbstractNamespace,
								},
							},
						},
					},
				},
			},
			wantErrs: semantic.UnsyncableResourcesInNonLeaf(&ast.TreeNode{}),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			errs := Inheritance(tc.objs)
			if !errors.Is(errs, tc.wantErrs) {
				t.Errorf("got Inheritance() error %v, want %v", errs, tc.wantErrs)
			}
		})
	}
}
