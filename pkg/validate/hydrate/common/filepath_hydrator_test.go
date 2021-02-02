package common

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/ast/node"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/testing/fake"
	"github.com/google/nomos/pkg/validate/parsed"
)

const dir = "acme/"

func TestFilepathTreeHydrator(t *testing.T) {
	testCases := []struct {
		name string
		root *parsed.TreeRoot
		want *parsed.TreeRoot
	}{
		{
			name: "ignore clusterregistry and system",
			root: &parsed.TreeRoot{
				ClusterRegistryObjects: []ast.FileObject{
					fake.Cluster(core.Name("prod-cluster")),
				},
				SystemObjects: []ast.FileObject{
					fake.Repo(),
				},
			},
			want: &parsed.TreeRoot{
				ClusterRegistryObjects: []ast.FileObject{
					fake.Cluster(core.Name("prod-cluster")),
				},
				SystemObjects: []ast.FileObject{
					fake.Repo(),
				},
			},
		},
		{
			name: "annotate resources",
			root: &parsed.TreeRoot{
				ClusterObjects: []ast.FileObject{
					fake.ClusterRoleAtPath("cluster/clusterrole.yaml", core.Name("reader")),
				},
				Tree: &ast.TreeNode{
					Relative: cmpath.RelativeSlash("namespaces"),
					Type:     node.AbstractNamespace,
					Objects: []*ast.NamespaceObject{
						{FileObject: fake.RoleAtPath("namespaces/role.yaml", core.Name("writer"))},
					},
					Children: []*ast.TreeNode{
						{
							Relative: cmpath.RelativeSlash("namespaces/hello"),
							Type:     node.Namespace,
							Objects: []*ast.NamespaceObject{
								{FileObject: fake.Namespace("namespaces/hello")},
								{FileObject: fake.RoleBindingAtPath("namespaces/hello/binding.yaml", core.Name("bind-writer"))},
							},
						},
					},
				},
			},
			want: &parsed.TreeRoot{
				ClusterObjects: []ast.FileObject{
					fake.ClusterRoleAtPath("cluster/clusterrole.yaml",
						core.Name("reader"),
						core.Annotation(v1.SourcePathAnnotationKey, dir+"cluster/clusterrole.yaml")),
				},
				Tree: &ast.TreeNode{
					Relative: cmpath.RelativeSlash("namespaces"),
					Type:     node.AbstractNamespace,
					Objects: []*ast.NamespaceObject{
						{FileObject: fake.RoleAtPath("namespaces/role.yaml",
							core.Name("writer"),
							core.Annotation(v1.SourcePathAnnotationKey, dir+"namespaces/role.yaml"))},
					},
					Children: []*ast.TreeNode{
						{
							Relative: cmpath.RelativeSlash("namespaces/hello"),
							Type:     node.Namespace,
							Objects: []*ast.NamespaceObject{
								{FileObject: fake.Namespace("namespaces/hello",
									core.Annotation(v1.SourcePathAnnotationKey, dir+"namespaces/hello/namespace.yaml"))},
								{FileObject: fake.RoleBindingAtPath("namespaces/hello/binding.yaml",
									core.Name("bind-writer"),
									core.Annotation(v1.SourcePathAnnotationKey, dir+"namespaces/hello/binding.yaml"))},
							},
						},
					},
				},
			},
		},
		{
			name: "preserve existing annotations",
			root: &parsed.TreeRoot{
				ClusterObjects: []ast.FileObject{
					fake.ClusterRoleAtPath("cluster/clusterrole.yaml",
						core.Name("reader"),
						core.Annotation("color", "blue")),
				},
			},
			want: &parsed.TreeRoot{
				ClusterObjects: []ast.FileObject{
					fake.ClusterRoleAtPath("cluster/clusterrole.yaml",
						core.Name("reader"),
						core.Annotation("color", "blue"),
						core.Annotation(v1.SourcePathAnnotationKey, dir+"cluster/clusterrole.yaml")),
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			h := FilepathTreeHydrator(cmpath.RelativeSlash(dir))
			if err := h.Hydrate(tc.root); err != nil {
				t.Errorf("Got Hydrate() error %v, want nil", err)
			}
			if tc.want != nil {
				if diff := cmp.Diff(tc.want, tc.root); diff != "" {
					t.Error(diff)
				}
			}
		})
	}
}
