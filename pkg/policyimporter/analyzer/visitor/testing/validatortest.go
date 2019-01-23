package testing

import (
	"testing"

	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
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
			object := &ast.NamespaceObject{FileObject: tc.Object}
			validator.VisitObject(object)

			if tc.ShouldFail {
				vettesting.ExpectErrors([]string{vt.ErrorCode}, validator.Error(), t)
			} else {
				vettesting.ExpectErrors(nil, validator.Error(), t)
			}
		})
	}
}

// NodeObjectsValidatorTestCase defines objects in a single TreeNode, and whether the collection of
// objects fails validation.
type NodeObjectsValidatorTestCase struct {
	Name       string
	ShouldFail bool
	// Objects is a collection of FileObjects all in the same TreeNode.
	Objects []ast.FileObject
}

// NodeObjectsValidatorTest defines a Validator which is initialized and run on each of the provided
// test cases.
type NodeObjectsValidatorTest struct {
	// Validator is the function which produces a fresh ValidatorVisitor.
	Validator func() *visitor.ValidatorVisitor
	// ErrorCode is what the Validator returns if there is an error.
	ErrorCode string
	// Test
	TestCases []NodeObjectsValidatorTestCase
}

// Run executes each test case in the test.
func (vt *NodeObjectsValidatorTest) Run(t *testing.T) {
	for _, tc := range vt.TestCases {
		t.Run(tc.Name, func(t *testing.T) {
			validator := vt.Validator()

			node := &ast.TreeNode{
				Objects: make([]*ast.NamespaceObject, len(tc.Objects)),
			}
			for i, object := range tc.Objects {
				node.Objects[i] = &ast.NamespaceObject{FileObject: object}
			}

			validator.VisitTreeNode(node)

			if tc.ShouldFail {
				vettesting.ExpectErrors([]string{vt.ErrorCode}, validator.Error(), t)
			} else {
				vettesting.ExpectErrors(nil, validator.Error(), t)
			}
		})
	}
}
