package system

import (
	"testing"

	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	vt "github.com/google/nomos/pkg/policyimporter/analyzer/visitor/testing"
	"github.com/google/nomos/pkg/testing/fake"
)

func TestKindValidator(t *testing.T) {
	test := vt.ObjectValidatorTest{
		Validator: NewKindValidator,
		ErrorCode: vet.IllegalKindInSystemErrorCode,
		TestCases: []vt.ObjectValidatorTestCase{
			{
				Name:   "Repo passes",
				Object: fake.Repo("system/repo.yaml"),
			},
			{
				Name:   "Sync passes",
				Object: fake.Sync("system/sync.yaml"),
			},
			{
				Name:   "HierarchyConfig passes",
				Object: fake.HierarchyConfig("system/hc.yaml"),
			},
			{
				Name:       "Role fails",
				Object:     fake.Role("system/role.yaml"),
				ShouldFail: true,
			},
		},
	}

	test.RunAll(t)
}
