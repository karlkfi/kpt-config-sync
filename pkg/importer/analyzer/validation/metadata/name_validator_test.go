package metadata

import (
	"testing"

	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	visitortesting "github.com/google/nomos/pkg/importer/analyzer/visitor/testing"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/testing/fake"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func fakeNamedObject(gvk schema.GroupVersionKind, name string) ast.FileObject {
	return fake.UnstructuredAtPath(gvk, "namespaces/role.yaml", core.Name(name))
}

func TestTopLevelNamespaceValidation(t *testing.T) {
	test := visitortesting.ObjectValidatorTest{
		Validator: NewNamespaceDirectoryNameValidator,
		ErrorCode: IllegalTopLevelNamespaceErrorCode,
		TestCases: []visitortesting.ObjectValidatorTestCase{
			{
				Name:       "illegal top level Namespace",
				Object:     fakeNamedObject(kinds.Namespace(), "Name"),
				ShouldFail: true,
			},
			{
				Name:   "legal top level non-Namespace",
				Object: fakeNamedObject(kinds.Role(), "name"),
			},
		},
	}

	test.RunAll(t)
}
