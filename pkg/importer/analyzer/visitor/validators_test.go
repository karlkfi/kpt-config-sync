package visitor_test

import (
	"testing"

	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/testing/fake"

	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/visitor"
	visitortesting "github.com/google/nomos/pkg/importer/analyzer/visitor/testing"
	"github.com/google/nomos/pkg/kinds"
)

func newFailAll() *visitor.ValidatorVisitor {
	return visitor.NewAllObjectValidator(func(o ast.FileObject) status.MultiError {
		return status.InternalError("test error")
	})
}

func TestNewAllObjectValidator(t *testing.T) {
	test := visitortesting.ObjectValidatorTest{
		Validator: newFailAll,
		ErrorCode: status.InternalErrorCode,
		TestCases: []visitortesting.ObjectValidatorTestCase{
			{
				Name:       "ValidateSystemObject",
				Object:     fake.UnstructuredAtPath(kinds.Role(), "system/role.yaml"),
				ShouldFail: true,
			},
			{
				Name:       "ValidateClusterRegistryObject",
				Object:     fake.UnstructuredAtPath(kinds.Role(), "clusterregistry/role.yaml"),
				ShouldFail: true,
			},
			{
				Name:       "ValidateSystemObject",
				Object:     fake.UnstructuredAtPath(kinds.Role(), "cluster/role.yaml"),
				ShouldFail: true,
			},
			{
				Name:       "ValidateNamespaceObject",
				Object:     fake.UnstructuredAtPath(kinds.Role(), "namespaces/role.yaml"),
				ShouldFail: true,
			},
		},
	}

	test.RunAll(t)
}
