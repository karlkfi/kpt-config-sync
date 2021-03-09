package hydrate

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/testing/fake"
	"github.com/google/nomos/pkg/validate/objects"
	"github.com/google/nomos/pkg/webhook/configuration"
)

func TestDeclaredVersion(t *testing.T) {
	testCases := []struct {
		name string
		objs *objects.Raw
		want *objects.Raw
	}{
		{
			name: "v1 RoleBinding",
			objs: &objects.Raw{
				Objects: []ast.FileObject{
					fake.RoleBinding(),
				},
			},
			want: &objects.Raw{
				Objects: []ast.FileObject{
					fake.RoleBinding(core.Label(configuration.DeclaredVersionLabel, "v1")),
				},
			},
		},
		{
			name: "v1beta1 RoleBinding",
			objs: &objects.Raw{
				Objects: []ast.FileObject{
					fake.RoleBindingV1Beta1(),
				},
			},
			want: &objects.Raw{
				Objects: []ast.FileObject{
					fake.RoleBindingV1Beta1(core.Label(configuration.DeclaredVersionLabel, "v1beta1")),
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			errs := DeclaredVersion(tc.objs)
			if errs != nil {
				t.Errorf("Got DeclaredVersion() error %v, want nil", errs)
			}
			if diff := cmp.Diff(tc.want, tc.objs, ast.CompareFileObject); diff != "" {
				t.Error(diff)
			}
		})
	}
}
