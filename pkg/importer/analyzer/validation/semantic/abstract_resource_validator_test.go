package semantic

import (
	"testing"

	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/vet"
	vt "github.com/google/nomos/pkg/importer/analyzer/visitor/testing"
	"github.com/google/nomos/pkg/testing/fake"
)

func TestAbstractResourceValidator(t *testing.T) {
	test := vt.ObjectsValidatorTest{
		Validator: NewAbstractResourceValidator,
		ErrorCode: vet.UnsyncableResourcesErrorCode,
		TestCases: []vt.ObjectsValidatorTestCase{
			{
				Name: "Empty is valid",
			},
			{
				Name: "Namespace without resources is valid",
				Objects: []ast.FileObject{
					fake.Namespace("namespaces/foo/ns.yaml"),
				},
			},
			{
				Name: "Empty Namespace with resource is valid",
				Objects: []ast.FileObject{
					fake.Namespace("namespaces/foo/ns.yaml"),
					fake.Role("namespaces/foo/role.yaml"),
				},
			},
			{
				Name: "Abstract Namespace without resource is valid",
				Objects: []ast.FileObject{
					fake.NamespaceSelector("namespaces/foo/nsel.yaml"),
				},
			},
			{
				Name: "Abstract Namespace with resource is invalid",
				Objects: []ast.FileObject{
					fake.Role("namespaces/foo/role.yaml"),
				},
				ShouldFail: true,
			},
			{
				Name: "Abstract Namespace with resource and Namespace child is valid",
				Objects: []ast.FileObject{
					fake.Role("namespaces/foo/role.yaml"),
					fake.Namespace("namespaces/foo/bar/ns.yaml"),
				},
			},
			{
				Name: "Abstract Namespace with resource and Abstract Namespace child is invalid",
				Objects: []ast.FileObject{
					fake.Role("namespaces/foo/role.yaml"),
					fake.NamespaceSelector("namespaces/foo/bar/nsel.yaml"),
				},
				ShouldFail: true,
			},
			{
				Name: "Abstract Namespace with resource, Abstract Namespace child, and Namespace child is valid",
				Objects: []ast.FileObject{
					fake.Role("namespaces/foo/role.yaml"),
					fake.Namespace("namespaces/foo/bar/ns.yaml"),
					fake.NamespaceSelector("namespaces/foo/baz/nsel.yaml"),
				},
			},
		},
	}

	test.RunAll(t)
}
