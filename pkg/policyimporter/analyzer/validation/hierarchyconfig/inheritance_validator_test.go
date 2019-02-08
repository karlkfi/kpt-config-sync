package hierarchyconfig

import (
	"testing"

	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	visitortesting "github.com/google/nomos/pkg/policyimporter/analyzer/visitor/testing"
	"github.com/google/nomos/pkg/testing/fake"
)

func hierarchyConfigMode(group, kind string, mode v1alpha1.HierarchyModeType) *v1alpha1.HierarchyConfig {
	return &v1alpha1.HierarchyConfig{
		Spec: v1alpha1.HierarchyConfigSpec{
			Resources: []v1alpha1.HierarchyConfigResource{
				{
					Group:         group,
					Kinds:         []string{kind},
					HierarchyMode: mode,
				},
			},
		},
	}
}

func TestInheritanceValidator(t *testing.T) {
	test := visitortesting.ObjectValidatorTest{
		Validator: NewInheritanceValidator,
		ErrorCode: vet.IllegalHierarchyModeErrorCode,
		TestCases: []visitortesting.ObjectValidatorTestCase{
			{
				Name: "inheritance rolebinding default",
				Object: fake.HierarchyConfigSpecified(
					"system/hc.yaml",
					hierarchyConfigMode(
						kinds.RoleBinding().Group,
						kinds.RoleBinding().Kind,
						v1alpha1.HierarchyModeDefault,
					),
				),
			},
			{
				Name: "inheritance rolebinding quota error",
				Object: fake.HierarchyConfigSpecified(
					"system/hc.yaml",
					hierarchyConfigMode(
						kinds.RoleBinding().Group,
						kinds.RoleBinding().Kind,
						v1alpha1.HierarchyModeHierarchicalQuota,
					),
				),
				ShouldFail: true,
			},
			{
				Name: "inheritance rolebinding inherit",
				Object: fake.HierarchyConfigSpecified(
					"system/hc.yaml",
					hierarchyConfigMode(
						kinds.RoleBinding().Group,
						kinds.RoleBinding().Kind,
						v1alpha1.HierarchyModeInherit,
					),
				),
			},
			{
				Name: "inheritance rolebinding none",
				Object: fake.HierarchyConfigSpecified(
					"system/hc.yaml",
					hierarchyConfigMode(
						kinds.RoleBinding().Group,
						kinds.RoleBinding().Kind,
						v1alpha1.HierarchyModeNone,
					),
				),
			},
			{
				Name: "inheritance rolebinding default",
				Object: fake.HierarchyConfigSpecified(
					"system/hc.yaml",
					hierarchyConfigMode(
						kinds.RoleBinding().Group,
						kinds.RoleBinding().Kind,
						v1alpha1.HierarchyModeDefault,
					),
				),
			},
			{
				Name: "inheritance resourcequota default",
				Object: fake.HierarchyConfigSpecified(
					"system/hc.yaml",
					hierarchyConfigMode(
						kinds.ResourceQuota().Group,
						kinds.ResourceQuota().Kind,
						v1alpha1.HierarchyModeDefault,
					),
				),
			},
			{
				Name: "inheritance resourcequota",
				Object: fake.HierarchyConfigSpecified(
					"system/hc.yaml",
					hierarchyConfigMode(
						kinds.ResourceQuota().Group,
						kinds.ResourceQuota().Kind,
						v1alpha1.HierarchyModeHierarchicalQuota,
					),
				),
			},
			{
				Name: "inheritance resourcequota inherit",
				Object: fake.HierarchyConfigSpecified(
					"system/hc.yaml",
					hierarchyConfigMode(
						kinds.ResourceQuota().Group,
						kinds.ResourceQuota().Kind,
						v1alpha1.HierarchyModeInherit,
					),
				),
			},
			{
				Name: "inheritance resourcequota none",
				Object: fake.HierarchyConfigSpecified(
					"system/hc.yaml",
					hierarchyConfigMode(
						kinds.ResourceQuota().Group,
						kinds.ResourceQuota().Kind,
						v1alpha1.HierarchyModeNone,
					),
				),
			},
		},
	}

	test.RunAll(t)
}
