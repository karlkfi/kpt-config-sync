package hydrate

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/metadata"
	"github.com/google/nomos/pkg/testing/fake"
	"github.com/google/nomos/pkg/validate/objects"
)

func TestClusterName(t *testing.T) {
	testCases := []struct {
		name string
		objs *objects.Raw
		want *objects.Raw
	}{
		{
			name: "Hydrate with cluster name",
			objs: &objects.Raw{
				ClusterName: "hello-world",
				Objects: []ast.FileObject{
					fake.ClusterRoleAtPath("cluster/clusterrole.yaml"),
					fake.RoleBindingAtPath("namespaces/foo/rolebinding.yaml"),
				},
			},
			want: &objects.Raw{
				ClusterName: "hello-world",
				Objects: []ast.FileObject{
					fake.ClusterRoleAtPath("cluster/clusterrole.yaml",
						core.Annotation(metadata.ClusterNameAnnotationKey, "hello-world")),
					fake.RoleBindingAtPath("namespaces/foo/rolebinding.yaml",
						core.Annotation(metadata.ClusterNameAnnotationKey, "hello-world")),
				},
			},
		},
		{
			name: "Hydrate with empty cluster name",
			objs: &objects.Raw{
				Objects: []ast.FileObject{
					fake.ClusterRoleAtPath("cluster/clusterrole.yaml"),
					fake.RoleBindingAtPath("namespaces/foo/rolebinding.yaml"),
				},
			},
			want: &objects.Raw{
				Objects: []ast.FileObject{
					fake.ClusterRoleAtPath("cluster/clusterrole.yaml"),
					fake.RoleBindingAtPath("namespaces/foo/rolebinding.yaml"),
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			errs := ClusterName(tc.objs)
			if errs != nil {
				t.Errorf("Got ClusterName() error %v, want nil", errs)
			}
			if diff := cmp.Diff(tc.want, tc.objs, ast.CompareFileObject); diff != "" {
				t.Error(diff)
			}
		})
	}
}
