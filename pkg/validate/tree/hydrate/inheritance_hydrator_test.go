package hydrate

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
	"github.com/google/nomos/pkg/validate/objects"
)

func TestInheritance(t *testing.T) {
	testCases := []struct {
		name     string
		objs     *objects.Tree
		want     *objects.Tree
		wantErrs status.MultiError
	}{
		{
			name: "Preserve non-namespace objects",
			objs: &objects.Tree{
				Repo: fake.Repo(),
				Cluster: []ast.FileObject{
					fake.ClusterRole(core.Name("hello-reader")),
					fake.ClusterRoleBinding(core.Name("hello-binding")),
				},
			},
			want: &objects.Tree{
				Repo: fake.Repo(),
				Cluster: []ast.FileObject{
					fake.ClusterRole(core.Name("hello-reader")),
					fake.ClusterRoleBinding(core.Name("hello-binding")),
				},
			},
		},
		{
			name: "Propagate abstract namespace objects",
			objs: &objects.Tree{
				Repo: fake.Repo(),
				HierarchyConfigs: []ast.FileObject{
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
			want: &objects.Tree{
				Repo: fake.Repo(),
				HierarchyConfigs: []ast.FileObject{
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
			wantErrs: status.Append(
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
			objs: &objects.Tree{
				Repo: fake.Repo(),
				HierarchyConfigs: []ast.FileObject{
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
			wantErrs: validation.IllegalAbstractNamespaceObjectKindError(fake.RoleAtPath("namespaces/hello/role.yaml", core.Name("writer"))),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			errs := Inheritance(tc.objs)
			if !errors.Is(errs, tc.wantErrs) {
				t.Errorf("Got Inheritance() error %v, want %v", errs, tc.wantErrs)
			}
			if tc.want != nil {
				if diff := cmp.Diff(tc.want, tc.objs, ast.CompareFileObject); diff != "" {
					t.Error(diff)
				}
			}
		})
	}
}
