package hydrate

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	oldhnc "github.com/google/nomos/pkg/importer/analyzer/hnc"
	"github.com/google/nomos/pkg/testing/fake"
	"github.com/google/nomos/pkg/validate/objects"
)

func TestBuilderVisitor(t *testing.T) {
	testCases := []struct {
		name string
		objs *objects.Raw
		want *objects.Raw
	}{
		{
			name: "label and annotate namespace",
			objs: &objects.Raw{
				Objects: []ast.FileObject{
					fake.Namespace("namespaces/foo/bar"),
					fake.Namespace("namespaces/qux"),
					fake.Role(),
				},
			},
			want: &objects.Raw{
				Objects: []ast.FileObject{
					fake.Namespace("namespaces/foo/bar",
						core.Annotation(oldhnc.AnnotationKeyV1A1, v1.ManagedByValue),
						core.Annotation(oldhnc.AnnotationKeyV1A2, v1.ManagedByValue),
						core.Label("foo.tree.hnc.x-k8s.io/depth", "1"),
						core.Label("bar.tree.hnc.x-k8s.io/depth", "0")),
					fake.Namespace("namespaces/qux",
						core.Annotation(oldhnc.AnnotationKeyV1A1, v1.ManagedByValue),
						core.Annotation(oldhnc.AnnotationKeyV1A2, v1.ManagedByValue),
						core.Label("qux.tree.hnc.x-k8s.io/depth", "0")),
					fake.Role(),
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if err := HNCDepth(tc.objs); err != nil {
				t.Errorf("Got HNCDepth() error %v, want nil", err)
			}
			if diff := cmp.Diff(tc.want, tc.objs, ast.CompareFileObject); diff != "" {
				t.Error(diff)
			}
		})
	}
}
