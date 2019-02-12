package hierarchyconfig

import (
	"testing"

	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	visitortesting "github.com/google/nomos/pkg/policyimporter/analyzer/visitor/testing"
	"github.com/google/nomos/pkg/testing/fake"
	"github.com/google/nomos/pkg/util/discovery"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func toAPIInfo(known ...schema.GroupVersionKind) (*discovery.APIInfo, error) {
	resources := make([]*metav1.APIResourceList, len(known))

	for i, gvk := range known {
		resources[i] = &metav1.APIResourceList{
			GroupVersion: gvk.GroupVersion().String(),
			APIResources: []metav1.APIResource{{Kind: gvk.Kind}},
		}
	}

	return discovery.NewAPIInfo(resources)
}

func TestKnownResourceValidator(t *testing.T) {
	apiInfo, err := toAPIInfo(kinds.RoleBinding())
	if err != nil {
		t.Fatalf("unexpected error forming APIInfo: %v", err)
	}

	test := visitortesting.ObjectValidatorTest{
		Validator: NewKnownResourceValidator,
		ErrorCode: vet.UnknownResourceInHierarchyConfigErrorCode,
		TestCases: []visitortesting.ObjectValidatorTestCase{
			{
				Name:    "ResourceQuota throws error if not known",
				APIInfo: apiInfo,
				Object: fake.HierarchyConfigSpecified(
					"system/hc.yaml",
					hierarchyConfig(
						kinds.ResourceQuota().Group,
						kinds.ResourceQuota().Kind,
					),
				),
				ShouldFail: true,
			},
			{
				Name:    "RoleBinding valid if known",
				APIInfo: apiInfo,
				Object: fake.HierarchyConfigSpecified(
					"system/hc.yaml",
					hierarchyConfig(
						kinds.RoleBinding().Group,
						kinds.RoleBinding().Kind,
					),
				),
			},
		},
	}

	test.RunAll(t)
}
