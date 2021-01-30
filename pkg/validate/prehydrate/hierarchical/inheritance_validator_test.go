package hierarchical

import (
	"errors"
	"testing"

	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/ast/node"
	"github.com/google/nomos/pkg/importer/analyzer/validation/semantic"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/testing/fake"
	"github.com/google/nomos/pkg/validate/parsed"
)

func TestInheritanceValidator(t *testing.T) {
	testCases := []struct {
		name    string
		root    parsed.Root
		wantErr status.MultiError
	}{
		{
			name: "empty tree",
			root: &parsed.TreeRoot{
				Tree: &ast.TreeNode{
					Relative: cmpath.RelativeSlash("namespaces"),
					Type:     node.AbstractNamespace,
				},
			},
		},
		{
			name: "Namespace without resources",
			root: &parsed.TreeRoot{
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
			root: &parsed.TreeRoot{
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
			root: &parsed.TreeRoot{
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
			root: &parsed.TreeRoot{
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
			root: &parsed.TreeRoot{
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
			root: &parsed.TreeRoot{
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
			wantErr: semantic.UnsyncableResourcesInLeaf(&ast.TreeNode{}),
		},
		{
			name: "abstract namespace with resource and abstract child",
			root: &parsed.TreeRoot{
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
			wantErr: semantic.UnsyncableResourcesInNonLeaf(&ast.TreeNode{}),
		},
	}

	for _, tc := range testCases {
		iv := InheritanceValidator()
		t.Run(tc.name, func(t *testing.T) {

			err := iv(tc.root)
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("got InheritanceValidator() error %v, want %v", err, tc.wantErr)
			}
		})
	}
}
