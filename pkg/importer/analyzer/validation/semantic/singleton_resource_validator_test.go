package semantic

import (
	"testing"

	"github.com/google/nomos/pkg/importer/analyzer/ast"
	visitortesting "github.com/google/nomos/pkg/importer/analyzer/visitor/testing"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/testing/fake"
)

func TestUniqueNamespaceValidator(t *testing.T) {
	test := visitortesting.ObjectsValidatorTest{
		Validator: func() ast.Visitor { return NewSingletonResourceValidator(kinds.Namespace()) },
		ErrorCode: MultipleSingletonsErrorCode,
		TestCases: []visitortesting.ObjectsValidatorTestCase{
			{
				Name: "empty",
			},
			{
				Name:    "one namespace",
				Objects: []ast.FileObject{fake.Namespace("namespaces/bar")},
			},
			{
				Name: "two namespace same dir",
				Objects: []ast.FileObject{
					fake.Namespace("namespaces/bar"),
					fake.Namespace("namespaces/bar"),
				},
				ShouldFail: true,
			},
			{
				Name: "two namespace different dir",
				Objects: []ast.FileObject{
					fake.Namespace("namespaces/bar"),
					fake.Namespace("namespaces/foo"),
				},
			},
		},
	}

	test.RunAll(t)
}
