package system_test

import (
	"testing"

	"github.com/google/nomos/pkg/importer/analyzer/validation/system"
	"github.com/google/nomos/pkg/importer/analyzer/vet"
	vt "github.com/google/nomos/pkg/importer/analyzer/visitor/testing"
	"github.com/google/nomos/pkg/testing/fake"
)

func TestKindValidator(t *testing.T) {
	test := vt.ObjectValidatorTest{
		Validator: system.NewKindValidator,
		ErrorCode: vet.IllegalKindInSystemErrorCode,
		TestCases: []vt.ObjectValidatorTestCase{
			{
				Name:   "Repo passes",
				Object: fake.Repo(),
			},
			{
				Name:   "HierarchyConfig passes",
				Object: fake.HierarchyConfigAtPath("system/config.yaml"),
			},
			{
				Name:       "Sync fails",
				Object:     fake.SyncAtPath("system/sync.yaml"),
				ShouldFail: true,
			},
			{
				Name:   "HierarchyConfig passes",
				Object: fake.HierarchyConfigAtPath("system/hc.yaml"),
			},
			{
				Name:       "Role fails",
				Object:     fake.RoleAtPath("system/role.yaml"),
				ShouldFail: true,
			},
		},
	}

	test.RunAll(t)
}
