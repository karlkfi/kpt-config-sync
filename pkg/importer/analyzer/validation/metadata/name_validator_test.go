package metadata

import (
	"testing"

	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/ast/asttesting"
	visitortesting "github.com/google/nomos/pkg/importer/analyzer/visitor/testing"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/kinds"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func fakeNamedObject(gvk schema.GroupVersionKind, name string) ast.FileObject {
	object := asttesting.NewFakeObject(gvk)
	object.SetName(name)
	return ast.NewFileObject(
		object,
		cmpath.FromSlash("namespaces/role.yaml"),
	)
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
