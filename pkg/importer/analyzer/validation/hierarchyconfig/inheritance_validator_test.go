package hierarchyconfig

import (
	"testing"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/testing/asttest"
	"github.com/google/nomos/pkg/testing/fake"
)

func TestInheritanceValidator(t *testing.T) {
	test := asttest.Validator(NewInheritanceValidator,
		IllegalHierarchyModeErrorCode,

		asttest.Pass("inheritance rolebinding default",
			fake.HierarchyConfig(
				fake.HierarchyConfigKind(v1.HierarchyModeDefault, kinds.RoleBinding())),
		),
		asttest.Pass("inheritance rolebinding inherit",
			fake.HierarchyConfig(
				fake.HierarchyConfigKind(v1.HierarchyModeInherit, kinds.RoleBinding())),
		),
		asttest.Pass("inheritance rolebinding none",
			fake.HierarchyConfig(
				fake.HierarchyConfigKind(v1.HierarchyModeNone, kinds.RoleBinding())),
		),
		asttest.Pass("inheritance resourcequota default",
			fake.HierarchyConfig(
				fake.HierarchyConfigKind(v1.HierarchyModeDefault, kinds.ResourceQuota())),
		),
		asttest.Pass("inheritance resourcequota inherit",
			fake.HierarchyConfig(
				fake.HierarchyConfigKind(v1.HierarchyModeInherit, kinds.ResourceQuota())),
		),
		asttest.Pass("inheritance resourcequota none",
			fake.HierarchyConfig(
				fake.HierarchyConfigKind(v1.HierarchyModeNone, kinds.ResourceQuota())),
		))

	test.RunAll(t)
}
