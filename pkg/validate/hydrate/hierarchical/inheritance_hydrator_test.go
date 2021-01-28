package hierarchical

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/ast/node"
	"github.com/google/nomos/pkg/importer/analyzer/validation"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/testing/fake"
	"github.com/google/nomos/pkg/validate/parsed"
)

func TestInheritanceHydrator_Hydrate(t *testing.T) {
	testCases := []struct {
		name    string
		root    *parsed.TreeRoot
		want    *parsed.TreeRoot
		wantErr status.MultiError
	}{
		{
			name: "Preserve non-namespace objects",
			root: &parsed.TreeRoot{
				ClusterRegistryObjects: []ast.FileObject{
					fake.Cluster(core.Name("prod-cluster")),
				},
				ClusterObjects: []ast.FileObject{
					fake.ClusterRole(core.Name("hello-reader")),
				},
				SystemObjects: []ast.FileObject{
					fake.Repo(),
				},
			},
			want: &parsed.TreeRoot{
				ClusterRegistryObjects: []ast.FileObject{
					fake.Cluster(core.Name("prod-cluster")),
				},
				ClusterObjects: []ast.FileObject{
					fake.ClusterRole(core.Name("hello-reader")),
				},
				SystemObjects: []ast.FileObject{
					fake.Repo(),
				},
			},
		},
		{
			name: "Propagate abstract namespace objects",
			root: &parsed.TreeRoot{
				SystemObjects: []ast.FileObject{
					fake.Repo(),
					fake.HierarchyConfig(
						fake.HierarchyConfigResource(v1.HierarchyModeDefault, kinds.Role().GroupVersion(), kinds.Role().Kind),
					),
				},
				Tree: &ast.TreeNode{
					Relative: cmpath.RelativeSlash("namespaces"),
					Type:     node.AbstractNamespace,
					Objects: []*ast.NamespaceObject{
						{FileObject: fake.RoleAtPath("namespaces/role.yaml", core.Name("reader"))},
					},
					Children: []*ast.TreeNode{
						{
							Relative: cmpath.RelativeSlash("namespaces/hello"),
							Type:     node.AbstractNamespace,
							Objects: []*ast.NamespaceObject{
								{FileObject: fake.RoleAtPath("namespaces/hello/role.yaml", core.Name("writer"))},
							},
							Children: []*ast.TreeNode{
								{
									Relative: cmpath.RelativeSlash("namespaces/hello/world"),
									Type:     node.Namespace,
									Objects: []*ast.NamespaceObject{
										{FileObject: fake.Namespace("namespaces/hello/world")},
									},
								},
								{
									Relative: cmpath.RelativeSlash("namespaces/hello/moon"),
									Type:     node.Namespace,
									Objects: []*ast.NamespaceObject{
										{FileObject: fake.Namespace("namespaces/hello/moon")},
										{FileObject: fake.Deployment("namespaces/hello/moon")},
									},
								},
							},
						},
						{
							Relative: cmpath.RelativeSlash("namespaces/goodbye"),
							Type:     node.Namespace,
							Objects: []*ast.NamespaceObject{
								{FileObject: fake.Namespace("namespaces/goodbye")},
								{FileObject: fake.Deployment("namespaces/goodbye")},
							},
						},
					},
				},
			},
			want: &parsed.TreeRoot{
				SystemObjects: []ast.FileObject{
					fake.Repo(),
					fake.HierarchyConfig(
						fake.HierarchyConfigResource(v1.HierarchyModeDefault, kinds.Role().GroupVersion(), kinds.Role().Kind),
					),
				},
				Tree: &ast.TreeNode{
					Relative: cmpath.RelativeSlash("namespaces"),
					Type:     node.AbstractNamespace,
					Objects: []*ast.NamespaceObject{
						{FileObject: fake.RoleAtPath("namespaces/role.yaml", core.Name("reader"))},
					},
					Children: []*ast.TreeNode{
						{
							Relative: cmpath.RelativeSlash("namespaces/hello"),
							Type:     node.AbstractNamespace,
							Objects: []*ast.NamespaceObject{
								{FileObject: fake.RoleAtPath("namespaces/hello/role.yaml", core.Name("writer"))},
							},
							Children: []*ast.TreeNode{
								{
									Relative: cmpath.RelativeSlash("namespaces/hello/world"),
									Type:     node.Namespace,
									Objects: []*ast.NamespaceObject{
										{FileObject: fake.Namespace("namespaces/hello/world")},
										{FileObject: fake.RoleAtPath("namespaces/role.yaml", core.Name("reader"))},
										{FileObject: fake.RoleAtPath("namespaces/hello/role.yaml", core.Name("writer"))},
									},
								},
								{
									Relative: cmpath.RelativeSlash("namespaces/hello/moon"),
									Type:     node.Namespace,
									Objects: []*ast.NamespaceObject{
										{FileObject: fake.Namespace("namespaces/hello/moon")},
										{FileObject: fake.Deployment("namespaces/hello/moon")},
										{FileObject: fake.RoleAtPath("namespaces/role.yaml", core.Name("reader"))},
										{FileObject: fake.RoleAtPath("namespaces/hello/role.yaml", core.Name("writer"))},
									},
								},
							},
						},
						{
							Relative: cmpath.RelativeSlash("namespaces/goodbye"),
							Type:     node.Namespace,
							Objects: []*ast.NamespaceObject{
								{FileObject: fake.Namespace("namespaces/goodbye")},
								{FileObject: fake.Deployment("namespaces/goodbye")},
								{FileObject: fake.RoleAtPath("namespaces/role.yaml", core.Name("reader"))},
							},
						},
					},
				},
			},
		},
		{
			name: "Validate Namespace can not have child Namespaces",
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
							Children: []*ast.TreeNode{
								{
									Relative: cmpath.RelativeSlash("namespaces/hello/world"),
									Type:     node.Namespace,
									Objects: []*ast.NamespaceObject{
										{FileObject: fake.Namespace("namespaces/hello/world")},
									},
								},
								{
									Relative: cmpath.RelativeSlash("namespaces/hello/moon"),
									Type:     node.Namespace,
									Objects: []*ast.NamespaceObject{
										{FileObject: fake.Namespace("namespaces/hello/moon")},
									},
								},
							},
						},
					},
				},
			},
			wantErr: status.Append(
				validation.IllegalNamespaceSubdirectoryError(
					&ast.TreeNode{Relative: cmpath.RelativeSlash("namespaces/hello/world")},
					&ast.TreeNode{Relative: cmpath.RelativeSlash("namespaces/hello")},
				),
				validation.IllegalNamespaceSubdirectoryError(
					&ast.TreeNode{Relative: cmpath.RelativeSlash("namespaces/hello/moon")},
					&ast.TreeNode{Relative: cmpath.RelativeSlash("namespaces/hello")},
				),
			),
		},
		{
			name: "Validate abstract namespace can not have invalid objects",
			root: &parsed.TreeRoot{
				SystemObjects: []ast.FileObject{
					fake.Repo(),
					fake.HierarchyConfig(
						fake.HierarchyConfigResource(v1.HierarchyModeNone, kinds.Role().GroupVersion(), kinds.Role().Kind),
					),
				},
				Tree: &ast.TreeNode{
					Relative: cmpath.RelativeSlash("namespaces"),
					Type:     node.AbstractNamespace,
					Children: []*ast.TreeNode{
						{
							Relative: cmpath.RelativeSlash("namespaces/hello"),
							Type:     node.AbstractNamespace,
							Objects: []*ast.NamespaceObject{
								{FileObject: fake.RoleAtPath("namespaces/hello/role.yaml", core.Name("writer"))},
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
			wantErr: validation.IllegalAbstractNamespaceObjectKindError(fake.RoleAtPath("namespaces/hello/role.yaml", core.Name("writer"))),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			h := InheritanceHydrator()
			err := h.Hydrate(tc.root)
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("Got Hydrate() error %v, want %v", err, tc.wantErr)
			}
			if tc.want != nil {
				if diff := cmp.Diff(tc.want, tc.root); diff != "" {
					t.Error(diff)
				}
			}
		})
	}
}
