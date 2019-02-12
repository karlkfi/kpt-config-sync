package testing

import (
	"testing"

	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/transform/tree/treetesting"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet/vettesting"
	"github.com/google/nomos/pkg/policyimporter/analyzer/visitor"
	"github.com/google/nomos/pkg/util/discovery"
)

// ObjectValidatorTestCase defines an individual FileObject to validate with the validator, and
// whether the object should fail.
type ObjectValidatorTestCase struct {
	Name       string
	ShouldFail bool
	Object     ast.FileObject
	APIInfo    *discovery.APIInfo
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

// RunAll executes each test case in the test.
func (vt *ObjectValidatorTest) RunAll(t *testing.T) {
	t.Helper()
	if vt.Validator == nil {
		t.Fatal("Assign a Validator factory method.")
	}

	for _, tc := range vt.TestCases {
		t.Run(tc.Name, func(t *testing.T) {
			t.Helper()

			validator := vt.Validator()
			var root *ast.Root
			if tc.APIInfo != nil {
				root = treetesting.BuildTreeWithAPIInfo(t, tc.APIInfo, tc.Object)
			} else {
				root = treetesting.BuildTree(t, tc.Object)
			}
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

// RunAll executes each test case in the test.
func (vt *ObjectsValidatorTest) RunAll(t *testing.T) {
	t.Helper()
	if vt.Validator == nil {
		t.Fatal("Assign a Validator factory method.")
	}

	for _, tc := range vt.TestCases {
		t.Run(tc.Name, func(t *testing.T) {
			t.Helper()

			validator := vt.Validator()

			root := treetesting.BuildTree(t, tc.Objects...)
			root.Accept(validator)

			if tc.ShouldFail {
				vettesting.ExpectErrors([]string{vt.ErrorCode}, validator.Error(), t)
			} else {
				vettesting.ExpectErrors(nil, validator.Error(), t)
			}
		})
	}
}
