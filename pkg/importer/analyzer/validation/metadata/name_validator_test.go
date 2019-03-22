package metadata

import (
	"testing"

	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/ast/asttesting"
	"github.com/google/nomos/pkg/importer/analyzer/vet"
	visitortesting "github.com/google/nomos/pkg/importer/analyzer/visitor/testing"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/kinds"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func fakeNamedObject(gvk schema.GroupVersionKind, name string) ast.FileObject {
	object := asttesting.NewFakeObject(gvk)
	object.SetName(name)
	return ast.FileObject{
		Path:   cmpath.FromSlash("namespaces/role.yaml"),
		Object: object,
	}
}

func TestNameExistenceValidation(t *testing.T) {
	test := visitortesting.ObjectValidatorTest{
		Validator: NewNameValidator,
		ErrorCode: vet.MissingObjectNameErrorCode,
		TestCases: []visitortesting.ObjectValidatorTestCase{
			{
				Name:       "empty name",
				Object:     fakeNamedObject(kinds.Role(), ""),
				ShouldFail: true,
			},
			{
				Name:   "legal name",
				Object: fakeNamedObject(kinds.Role(), "name"),
			},
		},
	}

	test.RunAll(t)
}

func TestCrdNameValidation(t *testing.T) {
	test := visitortesting.ObjectValidatorTest{
		Validator: NewNameValidator,
		ErrorCode: vet.InvalidMetadataNameErrorCode,
		TestCases: []visitortesting.ObjectValidatorTestCase{
			{
				Name:       "illegal crd name",
				Object:     fakeNamedObject(kinds.Cluster(), "Name"),
				ShouldFail: true,
			},
			{
				Name:   "legal crd name",
				Object: fakeNamedObject(kinds.Cluster(), "name"),
			},
			{
				Name:   "non crd with illegal crd name",
				Object: fakeNamedObject(kinds.ResourceQuota(), "Name"),
			},
		},
	}

	test.RunAll(t)
}

func TestTopLevelNamespaceValidation(t *testing.T) {
	test := visitortesting.ObjectValidatorTest{
		Validator: NewNameValidator,
		ErrorCode: vet.IllegalTopLevelNamespaceErrorCode,
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
