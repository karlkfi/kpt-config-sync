package validation

import (
	"testing"

	"github.com/google/nomos/bespin/pkg/kinds"
	nomoskinds "github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast/asttesting"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	vt "github.com/google/nomos/pkg/policyimporter/analyzer/visitor/testing"
)

func fakeIAM(version string, path string) ast.FileObject {
	return asttesting.NewFakeFileObject(kinds.IAMPolicy().WithVersion(version), path)
}

func TestUniqueIAMValidatorVisitTreeNode(t *testing.T) {
	test := vt.ObjectsValidatorTest{
		Validator: NewUniqueIAMValidator,
		ErrorCode: vet.UndocumentedErrorCode,
		TestCases: []vt.ObjectsValidatorTestCase{
			{
				Name: "empty",
			},
			{
				Name: "one IAMPolicy",
				Objects: []ast.FileObject{
					fakeIAM("v1", "hierarchy/foo/iam.yaml"),
				},
			},
			{
				Name: "one IAMPolicy and one Role",
				Objects: []ast.FileObject{
					fakeIAM("v1", "hierarchy/foo/iam.yaml"),
					asttesting.NewFakeFileObject(nomoskinds.Role(), "hierarchy/foo/role.yaml"),
				},
			},
			{
				Name: "two IAMPolicies same version",
				Objects: []ast.FileObject{
					fakeIAM("v1", "hierarchy/foo/iam-1.yaml"),
					fakeIAM("v1", "hierarchy/foo/iam-2.yaml"),
				},
				ShouldFail: true,
			},
			{
				Name: "two IAMPolicies different version",
				Objects: []ast.FileObject{
					fakeIAM("v1", "hierarchy/foo/iam-1.yaml"),
					fakeIAM("v2", "hierarchy/foo/iam-2.yaml"),
				},
				ShouldFail: true,
			},
			{
				Name: "two IAMPolicies different directories",
				Objects: []ast.FileObject{
					fakeIAM("v1", "hierarchy/foo/iam.yaml"),
					fakeIAM("v1", "hierarchy/bar/iam.yaml"),
				},
				ShouldFail: true,
			},
			{
				Name: "three IAMPolicies",
				Objects: []ast.FileObject{
					fakeIAM("v1", "hierarchy/foo/iam-1.yaml"),
					fakeIAM("v1", "hierarchy/foo/iam-2.yaml"),
					fakeIAM("v1", "hierarchy/foo/iam-3.yaml"),
				},
				ShouldFail: true,
			},
		},
	}

	test.Run(t)
}
