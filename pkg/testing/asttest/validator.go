package asttest

import (
	"testing"

	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/vet/vettesting"
)

// ValidatorTestCase defines an individual FileObject to validate with the validator, and
// whether the object should fail.
type ValidatorTestCase struct {
	// Name is the name of the test case.
	Name string
	// ShouldFail is true if validation is expected to fail.
	ShouldFail bool
	// Opts is the list of build options when constructing the AST for this test.
	Opts []ast.BuildOpt
}

// ValidatorTest defines a Validator which is initialized and run on each of the provided test
// cases.
type ValidatorTest struct {
	// Validator is the function which produces a fresh ValidatorVisitor.
	Validator func() ast.Visitor
	// ErrorCode is what the Validator returns if there is an error.
	ErrorCode string

	// TestCases is the list of test cases to run.
	TestCases []ValidatorTestCase

	// DefaultOpts is the set of build options when constructing the AST for all tests in this suite.
	// Runs before test-case-specific opts.
	DefaultOpts []ast.BuildOpt
}

// NewValidator is a function that produces a new visitor.
type NewValidator func() ast.Visitor

// Validator constructs a ValidatorTest.
// validator is the function to call to instantiate the validator.
// errorCode is the error code returned when validation does not pass.
// testCases is the set of test cases to run.
func Validator(validator NewValidator, errorCode string, testCases ...ValidatorTestCase) ValidatorTest {
	return ValidatorTest{
		Validator: validator,
		ErrorCode: errorCode,
		TestCases: testCases,
	}
}

// With adds BuildOpts to ValidatorTest, running before the construction of each AST.
func (vt ValidatorTest) With(opts ...ast.BuildOpt) ValidatorTest {
	return ValidatorTest{
		Validator:   vt.Validator,
		ErrorCode:   vt.ErrorCode,
		TestCases:   vt.TestCases,
		DefaultOpts: append(vt.DefaultOpts, opts...),
	}
}

// RunAll executes each test case in the test.
func (vt *ValidatorTest) RunAll(t *testing.T) {
	t.Helper()
	if vt.Validator == nil {
		t.Fatal("Assign a Validator factory method.")
	}

	for _, tc := range vt.TestCases {
		t.Run(tc.Name, func(t *testing.T) {
			t.Helper()

			validator := vt.Validator()
			opts := append(vt.DefaultOpts, tc.Opts...)
			root := Build(t, opts...)
			root.Accept(validator)

			if tc.ShouldFail {
				vettesting.ExpectErrors([]string{vt.ErrorCode}, validator.Error(), t)
			} else {
				vettesting.ExpectErrors(nil, validator.Error(), t)
			}
		})
	}
}

// Pass creates a ValidatorTestCase with a set of objects expected to pass validation.
func Pass(name string, objects ...ast.FileObject) ValidatorTestCase {
	return ValidatorTestCase{
		Name: name,
		Opts: []ast.BuildOpt{Objects(objects...)},
	}
}

// Fail creates a ValidatorTestCase with a set of objects expected to fail validation.
func Fail(name string, objects ...ast.FileObject) ValidatorTestCase {
	return ValidatorTestCase{
		Name:       name,
		ShouldFail: true,
		Opts:       []ast.BuildOpt{Objects(objects...)},
	}
}

// With appends additional BuildOpts to the ValidatorTestCase, allowing futher customization of the
// AST after ValidatorTest.DefaultOpts are run and objects are added.
func (tc ValidatorTestCase) With(opts ...ast.BuildOpt) ValidatorTestCase {
	return ValidatorTestCase{
		Name:       tc.Name,
		ShouldFail: tc.ShouldFail,
		Opts:       append(tc.Opts, opts...),
	}
}
