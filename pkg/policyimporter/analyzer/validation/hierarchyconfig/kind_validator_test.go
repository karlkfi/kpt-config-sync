package hierarchyconfig

import (
	"testing"

	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	testing2 "github.com/google/nomos/pkg/policyimporter/analyzer/visitor/testing"
	"github.com/google/nomos/pkg/testing/fake"
)

func hierarchyConfig(group string, kinds ...string) *v1alpha1.HierarchyConfig {
	return &v1alpha1.HierarchyConfig{
		Spec: v1alpha1.HierarchyConfigSpec{
			Resources: []v1alpha1.HierarchyConfigResource{
				{
					Group: group,
					Kinds: kinds,
				},
			},
		},
	}
}

func TestKindValidator(t *testing.T) {
	test := testing2.ObjectValidatorTest{
		Validator: NewHierarchyConfigKindValidator,
		ErrorCode: vet.UnsupportedResourceInHierarchyConfigErrorCode,
		TestCases: []testing2.ObjectValidatorTestCase{
			{
				Name:   "RoleBinding supported",
				Object: fake.HierarchyConfigSpecified("system/hc.yaml", hierarchyConfig("", "RoleBinding")),
			},
			{
				Name: "CRD not supported",
				Object: fake.HierarchyConfigSpecified(
					"system/hc.yaml",
					hierarchyConfig(
						kinds.CustomResourceDefinition().Group,
						kinds.CustomResourceDefinition().Kind),
				),
				ShouldFail: true,
			},
			{
				Name: "Namespace not supported",
				Object: fake.HierarchyConfigSpecified(
					"system/hc.yaml",
					hierarchyConfig(
						kinds.Namespace().Group,
						kinds.Namespace().Kind),
				),
				ShouldFail: true,
			},
			{
				Name: "omitting kind not supported",
				Object: fake.HierarchyConfigSpecified(
					"system/hc.yaml",
					hierarchyConfig(
						kinds.RoleBinding().Group,
					),
				),
				ShouldFail: true,
			},
			{
				Name: "nomos.dev group not supported",
				Object: fake.HierarchyConfigSpecified(
					"system/hc.yaml",
					hierarchyConfig(
						kinds.Sync().Group,
						kinds.Sync().Kind,
					),
				),
				ShouldFail: true,
			},
		},
	}

	test.RunAll(t)
}
