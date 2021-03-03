package hydrate

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/testing/fake"
	"github.com/google/nomos/pkg/validate/objects"
)

const dir = "acme/"

func TestFilepath(t *testing.T) {
	testCases := []struct {
		name string
		objs *objects.Raw
		want *objects.Raw
	}{
		{
			name: "Hydrate with filepaths",
			objs: &objects.Raw{
				PolicyDir: cmpath.RelativeSlash(dir),
				Objects: []ast.FileObject{
					fake.ClusterRoleAtPath("cluster/clusterrole.yaml", core.Name("reader")),
					fake.RoleAtPath("namespaces/role.yaml", core.Name("writer")),
					fake.Namespace("namespaces/hello"),
					fake.RoleBindingAtPath("namespaces/hello/binding.yaml", core.Name("bind-writer")),
				},
			},
			want: &objects.Raw{
				PolicyDir: cmpath.RelativeSlash(dir),
				Objects: []ast.FileObject{
					fake.ClusterRoleAtPath("cluster/clusterrole.yaml",
						core.Name("reader"),
						core.Annotation(v1.SourcePathAnnotationKey, dir+"cluster/clusterrole.yaml")),
					fake.RoleAtPath("namespaces/role.yaml",
						core.Name("writer"),
						core.Annotation(v1.SourcePathAnnotationKey, dir+"namespaces/role.yaml")),
					fake.Namespace("namespaces/hello",
						core.Annotation(v1.SourcePathAnnotationKey, dir+"namespaces/hello/namespace.yaml")),
					fake.RoleBindingAtPath("namespaces/hello/binding.yaml",
						core.Name("bind-writer"),
						core.Annotation(v1.SourcePathAnnotationKey, dir+"namespaces/hello/binding.yaml")),
				},
			},
		},
		{
			name: "Preserve existing annotations",
			objs: &objects.Raw{
				PolicyDir: cmpath.RelativeSlash(dir),
				Objects: []ast.FileObject{
					fake.ClusterRoleAtPath("cluster/clusterrole.yaml",
						core.Name("reader"),
						core.Annotation("color", "blue")),
				},
			},
			want: &objects.Raw{
				PolicyDir: cmpath.RelativeSlash(dir),
				Objects: []ast.FileObject{
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
			if err := Filepath(tc.objs); err != nil {
				t.Errorf("Got Filepath() error %v, want nil", err)
			}
			if diff := cmp.Diff(tc.want, tc.objs, ast.CompareFileObject); diff != "" {
				t.Error(diff)
			}
		})
	}
}
