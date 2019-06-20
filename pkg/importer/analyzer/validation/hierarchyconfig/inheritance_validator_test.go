package hierarchyconfig

import (
	"testing"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/vet"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/testing/asttest"
	"github.com/google/nomos/pkg/testing/fake"
)

func TestInheritanceValidator(t *testing.T) {
	test := asttest.Validator(NewInheritanceValidator,
		vet.IllegalHierarchyModeErrorCode,

		asttest.Pass("inheritance rolebinding default",
			fake.HierarchyConfig(
				fake.HierarchyConfigResource(kinds.RoleBinding(), v1.HierarchyModeDefault)),
		),
		asttest.Fail("inheritance rolebinding quota error",
			fake.HierarchyConfig(
				fake.HierarchyConfigResource(kinds.RoleBinding(), v1.HierarchyModeHierarchicalQuota)),
		),
		asttest.Pass("inheritance rolebinding inherit",
			fake.HierarchyConfig(
				fake.HierarchyConfigResource(kinds.RoleBinding(), v1.HierarchyModeInherit)),
		),
		asttest.Pass("inheritance rolebinding none",
			fake.HierarchyConfig(
				fake.HierarchyConfigResource(kinds.RoleBinding(), v1.HierarchyModeNone)),
		),
		asttest.Pass("inheritance resourcequota default",
			fake.HierarchyConfig(
				fake.HierarchyConfigResource(kinds.ResourceQuota(), v1.HierarchyModeDefault)),
		),
		asttest.Pass("inheritance resourcequota quota error",
			fake.HierarchyConfig(
				fake.HierarchyConfigResource(kinds.ResourceQuota(), v1.HierarchyModeHierarchicalQuota)),
		),
		asttest.Pass("inheritance resourcequota inherit",
			fake.HierarchyConfig(
				fake.HierarchyConfigResource(kinds.ResourceQuota(), v1.HierarchyModeInherit)),
		),
		asttest.Pass("inheritance resourcequota none",
			fake.HierarchyConfig(
				fake.HierarchyConfigResource(kinds.ResourceQuota(), v1.HierarchyModeNone)),
		))

	test.RunAll(t)
}
