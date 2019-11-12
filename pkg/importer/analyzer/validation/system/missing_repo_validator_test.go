package system_test

import (
	"testing"

	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/validation/system"
	vt "github.com/google/nomos/pkg/importer/analyzer/visitor/testing"
	"github.com/google/nomos/pkg/testing/fake"
)

func TestMissingRepoValidator(t *testing.T) {
	test := vt.ObjectsValidatorTest{
		Validator: system.NewMissingRepoValidator,
		ErrorCode: system.MissingRepoErrorCode,
		TestCases: []vt.ObjectsValidatorTestCase{
			{
				Name:       "No repo fails",
				ShouldFail: true,
			},
			{
				Name:    "Has repo passes",
				Objects: []ast.FileObject{fake.Repo()},
			},
		},
	}

	test.RunAll(t)
}
