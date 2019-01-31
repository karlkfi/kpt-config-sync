package validation

import (
	"testing"

	"github.com/google/nomos/pkg/bespin/kinds"
	nomoskinds "github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast/asttesting"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	vt "github.com/google/nomos/pkg/policyimporter/analyzer/visitor/testing"
)

func TestUniqueIAMValidatorVisitTreeNode(t *testing.T) {
	test := vt.NodeObjectsValidatorTest{
		Validator: NewUniqueIAMValidator,
		ErrorCode: vet.UndocumentedErrorCode,
		TestCases: []vt.NodeObjectsValidatorTestCase{
			{
				Name: "empty",
			},
			{
				Name: "one IAMPolicy",
				Objects: []ast.FileObject{
					asttesting.NewFakeFileObject(kinds.IAMPolicy().WithVersion(""), ""),
				},
			},
			{
				Name: "one IAMPolicy and one Role",
				Objects: []ast.FileObject{
					asttesting.NewFakeFileObject(kinds.IAMPolicy().WithVersion(""), ""),
					asttesting.NewFakeFileObject(nomoskinds.Role(), ""),
				},
			},
			{
				Name: "two IAMPolicies same version",
				Objects: []ast.FileObject{
					asttesting.NewFakeFileObject(kinds.IAMPolicy().WithVersion("v1"), ""),
					asttesting.NewFakeFileObject(kinds.IAMPolicy().WithVersion("v1"), ""),
				},
				ShouldFail: true,
			},
			{
				Name: "two IAMPolicies different version",
				Objects: []ast.FileObject{
					asttesting.NewFakeFileObject(kinds.IAMPolicy().WithVersion("v1"), ""),
					asttesting.NewFakeFileObject(kinds.IAMPolicy().WithVersion("v2"), ""),
				},
				ShouldFail: true,
			},
			{
				Name: "three IAMPolicies",
				Objects: []ast.FileObject{
					asttesting.NewFakeFileObject(kinds.IAMPolicy().WithVersion(""), ""),
					asttesting.NewFakeFileObject(kinds.IAMPolicy().WithVersion(""), ""),
					asttesting.NewFakeFileObject(kinds.IAMPolicy().WithVersion(""), ""),
				},
				ShouldFail: true,
			},
		},
	}

	test.Run(t)
}
