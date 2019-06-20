package visitors

import (
	"testing"

	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/vet"
	visitortesting "github.com/google/nomos/pkg/importer/analyzer/visitor/testing"
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
				Objects: []ast.FileObject{fake.RoleAtPath("namespaces/role.yaml")},
			},
			{
				Name:    "one dir",
				Objects: []ast.FileObject{fake.RoleAtPath("namespaces/foo/role.yaml")},
			},
			{
				Name:    "subdirectory of self",
				Objects: []ast.FileObject{fake.RoleAtPath("namespaces/foo/foo/role.yaml")},
			},
			{
				Name:    "deep subdirectory of self",
				Objects: []ast.FileObject{fake.RoleAtPath("namespaces/foo/bar/foo/role.yaml")},
			},
			{
				Name: "two leaf namespaces may not have the same name",
				Objects: []ast.FileObject{
					fake.Namespace("namespaces/bar/foo"),
					fake.Namespace("namespaces/qux/foo"),
					fake.RoleAtPath("namespaces/bar/foo/role.yaml"),
					fake.RoleAtPath("namespaces/qux/foo/role.yaml")},
				ShouldFail: true,
			},
			{
				Name: "an abstract namespace may have the short name as a leaf namespace",
				Objects: []ast.FileObject{
					//fake.Namespace("namespaces/foo"),
					fake.Namespace("namespaces/foo/foo"),
					fake.RoleAtPath("namespaces/foo/foo/role.yaml")},
			},
			{
				Name: "two directories corresponding to abstract namespaces",
				Objects: []ast.FileObject{
					fake.RoleAtPath("namespaces/foo/bar/baz/role.yaml"),
					fake.RoleAtPath("namespaces/qux/bar/quux/role.yaml")},
			},
			{
				Name: "directory with two children",
				Objects: []ast.FileObject{
					fake.RoleAtPath("namespaces/bar/foo/role.yaml"),
					fake.RoleAtPath("namespaces/bar/qux/role.yaml")},
			},
		},
	}

	test.RunAll(t)
}
