package nonhierarchical_test

import (
	"strings"
	"testing"

	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical"
	nht "github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical/nonhierarchicaltest"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/testing/fake"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func crd(name string, gvk schema.GroupVersionKind) ast.FileObject {
	result := fake.CustomResourceDefinitionV1Beta1Object()
	result.Name = name
	result.Spec.Group = gvk.Group
	result.Spec.Names = v1beta1.CustomResourceDefinitionNames{
		Plural: strings.ToLower(gvk.Kind) + "s",
		Kind:   gvk.Kind,
	}
	return fake.FileObject(result, "crd.yaml")
}

func TestDisallowedCRDNameValidator(t *testing.T) {
	testCases := []nht.ValidatorTestCase{
		// v1beta1
		nht.Pass("v1beta1 valid name",
			crd("anvils.acme.com", kinds.Anvil())),
		nht.Fail("v1beta1 non plural",
			crd("anvil.acme.com", kinds.Anvil())),
		nht.Fail("v1beta1 missing group",
			crd("anvils", kinds.Anvil())),
		// v1
		nht.Pass("v1 valid name",
			fake.ToCustomResourceDefinitionV1(crd("anvils.acme.com", kinds.Anvil()))),
		nht.Fail("v1 non plural",
			fake.ToCustomResourceDefinitionV1(crd("anvil.acme.com", kinds.Anvil()))),
		nht.Fail("v1 missing group",
			fake.ToCustomResourceDefinitionV1(crd("anvils", kinds.Anvil()))),
	}

	nht.RunAll(t, nonhierarchical.CRDNameValidator, testCases)
}
