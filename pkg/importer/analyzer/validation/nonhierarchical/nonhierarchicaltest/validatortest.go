package nonhierarchicaltest

import (
	"testing"

	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical"
)

// ValidatorTestCase is a test case for non-hierarchical validators.
type ValidatorTestCase struct {
	name       string
	objects    []ast.FileObject
	shouldFail bool
}

// Pass constructs a ValidatorTestCase that is expected to pass validation.
func Pass(name string, objects ...ast.FileObject) ValidatorTestCase {
	return ValidatorTestCase{
		name:       name,
		objects:    objects,
		shouldFail: false,
	}
}

// Fail cosntructs a ValidatorTestCase that is expected to fail validation.
func Fail(name string, objects ...ast.FileObject) ValidatorTestCase {
	return ValidatorTestCase{
		name:       name,
		objects:    objects,
		shouldFail: true,
	}
}

// RunAll runs all ValidatorTestCases with the passed Validator.
func RunAll(t *testing.T, validator nonhierarchical.Validator, tcs []ValidatorTestCase) {
	t.Helper()

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			errs := validator.Validate(tc.objects)
			if tc.shouldFail {
				if errs == nil {
					t.Fatal("expected error")
				}
			} else {
				if errs != nil {
					t.Fatal("unexpected error", errs)
				}
			}
		})
	}
}
