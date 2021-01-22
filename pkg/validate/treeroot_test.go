package validate

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/ast/node"
	"github.com/google/nomos/pkg/importer/analyzer/validation/system"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/testing/fake"
)

func TestBuildTree(t *testing.T) {
	testCases := []struct {
		name    string
		from    Root
		want    *TreeRoot
		wantErr status.MultiError
	}{
		{
			name: "almost-empty tree",
			from: &FlatRoot{
				SystemObjects: []ast.FileObject{
					fake.Repo(),
				},
			},
			want: &TreeRoot{
				SystemObjects: []ast.FileObject{
					fake.Repo(),
				},
			},
			wantErr: nil,
		},
		{
			name: "populated tree",
			from: &FlatRoot{
				ClusterRegistryObjects: []ast.FileObject{
					fake.Cluster(core.Name("prod-cluster")),
				},
				ClusterObjects: []ast.FileObject{
					fake.ClusterRole(core.Name("hello-reader")),
				},
				NamespaceObjects: []ast.FileObject{
					fake.Namespace("namespaces/hello/world"),
					fake.Namespace("namespaces/hello/moon"),
					fake.RoleAtPath("namespaces/hello/role.yaml", core.Name("writer")),
				},
				SystemObjects: []ast.FileObject{
					fake.Repo(),
				},
			},
			want: &TreeRoot{
				ClusterRegistryObjects: []ast.FileObject{
					fake.Cluster(core.Name("prod-cluster")),
				},
				ClusterObjects: []ast.FileObject{
					fake.ClusterRole(core.Name("hello-reader")),
				},
				SystemObjects: []ast.FileObject{
					fake.Repo(),
				},
				Tree: &ast.TreeNode{
					Relative: cmpath.RelativeSlash("namespaces"),
					Type:     node.AbstractNamespace,
					Children: []*ast.TreeNode{
						{
							Relative: cmpath.RelativeSlash("namespaces/hello"),
							Type:     node.AbstractNamespace,
							Objects: []*ast.NamespaceObject{
								{
									FileObject: fake.RoleAtPath("namespaces/hello/role.yaml", core.Name("writer")),
								},
							},
							Children: []*ast.TreeNode{
								{
									Relative: cmpath.RelativeSlash("namespaces/hello/moon"),
									Type:     node.Namespace,
									Objects: []*ast.NamespaceObject{
										{
											FileObject: fake.Namespace("namespaces/hello/moon"),
										},
									},
								},
								{
									Relative: cmpath.RelativeSlash("namespaces/hello/world"),
									Type:     node.Namespace,
									Objects: []*ast.NamespaceObject{
										{
											FileObject: fake.Namespace("namespaces/hello/world"),
										},
									},
								},
							},
						},
					},
				},
			},
			wantErr: nil,
		},
		{
			name: "missing Repo",
			from: &FlatRoot{
				ClusterObjects: []ast.FileObject{
					fake.ClusterRole(core.Name("hello")),
				},
			},
			want:    nil,
			wantErr: status.Append(nil, system.MissingRepoError()),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := BuildTree(tc.from)
			var gotErr, wantErr string
			if err != nil {
				gotErr = err.Error()
			}
			if tc.wantErr != nil {
				wantErr = tc.wantErr.Error()
			}
			if gotErr != wantErr {
				t.Errorf("Got BuildTree() error %v, want %v", gotErr, wantErr)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Error(diff)
			}
		})
	}
}
