package visitors

import (
	"testing"

	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	visitortesting "github.com/google/nomos/pkg/policyimporter/analyzer/visitor/testing"
	"github.com/google/nomos/pkg/testing/fake"
)

func TestUniqueDirectoryValidator(t *testing.T) {
	test := visitortesting.ObjectsValidatorTest{
		Validator: NewUniqueDirectoryValidator,
		ErrorCode: vet.DuplicateDirectoryNameErrorCode,
		TestCases: []visitortesting.ObjectsValidatorTestCase{
			{
				Name: "empty",
			},
			{
				Name:    "just namespaces/",
				Objects: []ast.FileObject{fake.Role("namespaces/role.yaml")},
			},
			{
				Name:    "one dir",
				Objects: []ast.FileObject{fake.Role("namespaces/foo/role.yaml")},
			},
			{
				Name:       "subdirectory of self",
				Objects:    []ast.FileObject{fake.Role("namespaces/foo/foo/role.yaml")},
				ShouldFail: true,
			},
			{
				Name:       "deep subdirectory of self",
				Objects:    []ast.FileObject{fake.Role("namespaces/foo/bar/foo/role.yaml")},
				ShouldFail: true,
			},
			{
				Name:       "child of different directories",
				Objects:    []ast.FileObject{fake.Role("namespaces/bar/foo/role.yaml"), fake.Role("namespaces/qux/foo/role.yaml")},
				ShouldFail: true,
			},
			{
				Name:    "directory with two children",
				Objects: []ast.FileObject{fake.Role("namespaces/bar/foo/role.yaml"), fake.Role("namespaces/bar/qux/role.yaml")},
			},
		},
	}

	test.RunAll(t)
}
