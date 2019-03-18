package system_test

import (
	"testing"

	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/validation/system"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	vt "github.com/google/nomos/pkg/policyimporter/analyzer/visitor/testing"
	"github.com/google/nomos/pkg/testing/fake"
)

func TestMissingRepoValidator(t *testing.T) {
	test := vt.ObjectsValidatorTest{
		Validator: system.NewMissingRepoValidator,
		ErrorCode: vet.MissingRepoErrorCode,
		TestCases: []vt.ObjectsValidatorTestCase{
			{
				Name:       "No repo fails",
				ShouldFail: true,
			},
			{
				Name:    "Has repo passes",
				Objects: []ast.FileObject{fake.Repo("system/repo.yaml")},
			},
		},
	}

	test.RunAll(t)
}
