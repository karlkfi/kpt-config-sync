package nonhierarchical_test

import (
	"strings"
	"testing"

	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/testing/fake"
	"github.com/pkg/errors"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func crdv1beta1(name string, gvk schema.GroupVersionKind) ast.FileObject {
	result := fake.CustomResourceDefinitionV1Beta1Object()
	result.Name = name
	result.Spec.Group = gvk.Group
	result.Spec.Names = apiextensionsv1beta1.CustomResourceDefinitionNames{
		Plural: strings.ToLower(gvk.Kind) + "s",
		Kind:   gvk.Kind,
	}
	return fake.FileObject(result, "crd.yaml")
}

func crdv1(name string, gvk schema.GroupVersionKind) ast.FileObject {
	result := fake.CustomResourceDefinitionV1Object()
	result.Name = name
	result.Spec.Group = gvk.Group
	result.Spec.Names = apiextensionsv1.CustomResourceDefinitionNames{
		Plural: strings.ToLower(gvk.Kind) + "s",
		Kind:   gvk.Kind,
	}
	return fake.FileObject(result, "crd.yaml")
}

func TestValidateCRDName(t *testing.T) {
	testCases := []struct {
		name string
		obj  ast.FileObject
		want status.Error
	}{
		// v1beta1 CRDs
		{
			name: "v1beta1 valid name",
			obj:  crdv1beta1("anvils.acme.com", kinds.Anvil()),
		},
		{
			name: "v1beta1 non plural",
			obj:  crdv1beta1("anvil.acme.com", kinds.Anvil()),
			want: fake.Error(nonhierarchical.InvalidCRDNameErrorCode),
		},
		{
			name: "v1beta1 missing group",
			obj:  crdv1beta1("anvils", kinds.Anvil()),
			want: fake.Error(nonhierarchical.InvalidCRDNameErrorCode),
		},
		// v1 CRDs
		{
			name: "v1 valid name",
			obj:  crdv1("anvils.acme.com", kinds.Anvil()),
		},
		{
			name: "v1 non plural",
			obj:  crdv1("anvil.acme.com", kinds.Anvil()),
			want: fake.Error(nonhierarchical.InvalidCRDNameErrorCode),
		},
		{
			name: "v1 missing group",
			obj:  crdv1("anvils", kinds.Anvil()),
			want: fake.Error(nonhierarchical.InvalidCRDNameErrorCode),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := nonhierarchical.ValidateCRDName(tc.obj)
			if !errors.Is(err, tc.want) {
				t.Errorf("got ValidateCRDName() error %v, want %v", err, tc.want)
			}
		})
	}
}
