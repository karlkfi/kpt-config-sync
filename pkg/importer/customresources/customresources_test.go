package customresources

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/testing/fake"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func groupKind(t *testing.T, gk schema.GroupKind) core.MetaMutator {
	return func(o core.Object) {
		crd, ok := o.(*v1beta1.CustomResourceDefinition)
		if !ok {
			t.Fatalf("not a v1beta1.CRD: %T", o)
		}
		crd.Spec.Group = gk.Group
		crd.Spec.Names.Kind = gk.Kind
	}
}

func servedStorage(t *testing.T, served, storage bool) core.MetaMutator {
	return func(o core.Object) {
		crd, ok := o.(*v1beta1.CustomResourceDefinition)
		if !ok {
			t.Fatalf("not a v1beta1.CRD: %T", o)
		}
		crd.Spec.Versions = []v1beta1.CustomResourceDefinitionVersion{
			{
				Served:  served,
				Storage: storage,
			},
		}
	}
}

func TestGetCRDs(t *testing.T) {
	testCases := []struct {
		name string
		objs []ast.FileObject
		want []*v1beta1.CustomResourceDefinition
	}{
		{
			name: "empty is fine",
		},
		{
			name: "ignore non-CRD",
			objs: []ast.FileObject{fake.Role()},
		},
		{
			name: "one v1beta1 CRD",
			objs: []ast.FileObject{
				fake.CustomResourceDefinitionV1Beta1(),
			},
			want: []*v1beta1.CustomResourceDefinition{
				fake.CustomResourceDefinitionV1Beta1Object(),
			},
		},
		{
			name: "one v1 CRD",
			objs: []ast.FileObject{
				fake.ToCustomResourceDefinitionV1(fake.CustomResourceDefinitionV1Beta1()),
			},
			want: []*v1beta1.CustomResourceDefinition{
				// The default if unspecified is true/true for served/storage.
				fake.CustomResourceDefinitionV1Beta1Object(servedStorage(t, true, true)),
			},
		},
		{
			name: "both CRD versions",
			objs: []ast.FileObject{
				fake.CustomResourceDefinitionV1Beta1(core.Name("a"),
					groupKind(t, kinds.Role().GroupKind())),
				fake.ToCustomResourceDefinitionV1(fake.CustomResourceDefinitionV1Beta1(
					core.Name("b"),
					groupKind(t, kinds.ClusterRole().GroupKind())),
				),
			},
			want: []*v1beta1.CustomResourceDefinition{
				// The default if unspecified is true/true for served/storage.
				fake.CustomResourceDefinitionV1Beta1Object(core.Name("a"),
					groupKind(t, kinds.Role().GroupKind())),
				fake.CustomResourceDefinitionV1Beta1Object(
					servedStorage(t, true, true),
					core.Name("b"),
					groupKind(t, kinds.ClusterRole().GroupKind()),
				),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := GetCRDs(tc.objs)

			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(tc.want, actual, cmpopts.EquateEmpty()); diff != "" {
				t.Error(diff)
			}
		})
	}
}
