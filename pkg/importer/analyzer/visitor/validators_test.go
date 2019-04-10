package visitor_test

import (
	"testing"

	"github.com/google/nomos/pkg/status"

	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/ast/asttesting"
	"github.com/google/nomos/pkg/importer/analyzer/vet"
	"github.com/google/nomos/pkg/importer/analyzer/visitor"
	visitortesting "github.com/google/nomos/pkg/importer/analyzer/visitor/testing"
	"github.com/google/nomos/pkg/kinds"
)

func newFailAll() *visitor.ValidatorVisitor {
	return visitor.NewAllObjectValidator(func(o ast.FileObject) status.MultiError {
		return status.From(vet.InternalError("test error"))
	})
}

func TestNewAllObjectValidator(t *testing.T) {
	test := visitortesting.ObjectValidatorTest{
		Validator: newFailAll,
		ErrorCode: vet.InternalErrorCode,
		TestCases: []visitortesting.ObjectValidatorTestCase{
			{
				Name:       "ValidateSystemObject",
				Object:     asttesting.NewFakeFileObject(kinds.Role(), "system/role.yaml"),
				ShouldFail: true,
			},
			{
				Name:       "ValidateClusterRegistryObject",
				Object:     asttesting.NewFakeFileObject(kinds.Role(), "clusterregistry/role.yaml"),
				ShouldFail: true,
			},
			{
				Name:       "ValidateSystemObject",
				Object:     asttesting.NewFakeFileObject(kinds.Role(), "cluster/role.yaml"),
				ShouldFail: true,
			},
			{
				Name:       "ValidateNamespaceObject",
				Object:     asttesting.NewFakeFileObject(kinds.Role(), "namespaces/role.yaml"),
				ShouldFail: true,
			},
		},
	}

	test.RunAll(t)
}
