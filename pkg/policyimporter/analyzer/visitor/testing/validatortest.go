package testing

import (
	"testing"

	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/transform/tree/treetesting"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet/vettesting"
	"github.com/google/nomos/pkg/policyimporter/analyzer/visitor"
)

// ObjectValidatorTestCase defines an individual FileObject to validate with the validator, and
// whether the object should fail.
type ObjectValidatorTestCase struct {
	Name       string
	ShouldFail bool
	Object     ast.FileObject
}

// ObjectValidatorTest defines a Validator which is initialized and run on each of the provided test
// cases.
type ObjectValidatorTest struct {
	// Validator is the function which produces a fresh ValidatorVisitor.
	Validator func() *visitor.ValidatorVisitor
	// ErrorCode is what the Validator returns if there is an error.
	ErrorCode string

	TestCases []ObjectValidatorTestCase
}

// Run executes each test case in the test.
func (vt *ObjectValidatorTest) Run(t *testing.T) {
	for _, tc := range vt.TestCases {
		t.Run(tc.Name, func(t *testing.T) {
			validator := vt.Validator()
			root := treetesting.BuildTree(tc.Object)
			root.Accept(validator)

			if tc.ShouldFail {
				vettesting.ExpectErrors([]string{vt.ErrorCode}, validator.Error(), t)
			} else {
				vettesting.ExpectErrors(nil, validator.Error(), t)
			}
		})
	}
}

// ObjectsValidatorTestCase defines objects in a repository, and whether the collection of
// objects fails validation.
type ObjectsValidatorTestCase struct {
	Name       string
	ShouldFail bool
	// Objects is a collection of FileObjects, not necessarily in the same TreeNode.
	Objects []ast.FileObject
}

// ObjectsValidatorTest defines a Validator which is initialized and run on each of the provided
// test cases.
type ObjectsValidatorTest struct {
	// Validator is the function which produces a fresh ValidatorVisitor.
	Validator func() ast.Visitor
	// ErrorCode is what the Validator returns if there is an error.
	ErrorCode string
	// Test
	TestCases []ObjectsValidatorTestCase
}

// Run executes each test case in the test.
func (vt *ObjectsValidatorTest) Run(t *testing.T) {
	for _, tc := range vt.TestCases {
		t.Run(tc.Name, func(t *testing.T) {
			validator := vt.Validator()

			root := treetesting.BuildTree(tc.Objects...)
			root.Accept(validator)

			if tc.ShouldFail {
				vettesting.ExpectErrors([]string{vt.ErrorCode}, validator.Error(), t)
			} else {
				vettesting.ExpectErrors(nil, validator.Error(), t)
			}
		})
	}
}
