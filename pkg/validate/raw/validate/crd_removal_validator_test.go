package validate

import (
	"errors"
	"testing"

	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/testing/fake"
	"github.com/google/nomos/pkg/validate/objects"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func crd(gvk schema.GroupVersionKind) *v1beta1.CustomResourceDefinition {
	return &v1beta1.CustomResourceDefinition{
		Spec: v1beta1.CustomResourceDefinitionSpec{
			Group: gvk.Group,
			Names: v1beta1.CustomResourceDefinitionNames{
				Kind: gvk.Kind,
			},
		},
	}
}

func crdFileObject(t *testing.T, gvk schema.GroupVersionKind) ast.FileObject {
	t.Helper()
	u := fake.CustomResourceDefinitionV1Beta1Unstructured()
	if err := unstructured.SetNestedField(u.Object, gvk.Group, "spec", "group"); err != nil {
		t.Fatal(err)
	}
	if err := unstructured.SetNestedField(u.Object, gvk.Kind, "spec", "names", "kind"); err != nil {
		t.Fatal(err)
	}
	return fake.FileObject(u, "crd.yaml")
}

func TestRemovedCRDs(t *testing.T) {
	testCases := []struct {
		name    string
		objs    *objects.Raw
		wantErr status.MultiError
	}{
		{
			name: "no previous or current CRDs",
			objs: &objects.Raw{
				Objects: []ast.FileObject{
					fake.AnvilAtPath("anvil1.yaml"),
				},
			},
		},
		{
			name: "add a CRD",
			objs: &objects.Raw{
				Objects: []ast.FileObject{
					crdFileObject(t, kinds.Anvil()),
					fake.AnvilAtPath("anvil1.yaml"),
				},
			},
		},
		{
			name: "keep a CRD",
			objs: &objects.Raw{
				PreviousCRDs: []*v1beta1.CustomResourceDefinition{
					crd(kinds.Anvil()),
				},
				Objects: []ast.FileObject{
					crdFileObject(t, kinds.Anvil()),
					fake.AnvilAtPath("anvil1.yaml"),
				},
			},
		},
		{
			name: "remove an unused CRD",
			objs: &objects.Raw{
				PreviousCRDs: []*v1beta1.CustomResourceDefinition{
					crd(kinds.Anvil()),
				},
				Objects: []ast.FileObject{
					fake.Role(),
				},
			},
		},
		{
			name: "remove an in-use CRD",
			objs: &objects.Raw{
				PreviousCRDs: []*v1beta1.CustomResourceDefinition{
					crd(kinds.Anvil()),
				},
				Objects: []ast.FileObject{
					fake.AnvilAtPath("anvil1.yaml"),
				},
			},
			wantErr: nonhierarchical.UnsupportedCRDRemovalError(fake.AnvilAtPath("anvil1.yaml")),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := RemovedCRDs(tc.objs)
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("got RemovedCRDs() error %v, want %v", err, tc.wantErr)
			}
		})
	}
}
