package semantic

import (
	"testing"

	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	visitortesting "github.com/google/nomos/pkg/policyimporter/analyzer/visitor/testing"
	"github.com/google/nomos/pkg/testing/fake"
)

func TestUniqueNamespaceValidator(t *testing.T) {
	test := visitortesting.ObjectsValidatorTest{
		Validator: func() ast.Visitor { return NewSingletonResourceValidator(kinds.Namespace()) },
		ErrorCode: vet.MultipleNamespacesErrorCode,
		TestCases: []visitortesting.ObjectsValidatorTestCase{
			{
				Name: "empty",
			},
			{
				Name:    "one namespace",
				Objects: []ast.FileObject{fake.Namespace("namespaces/bar/ns.yaml")},
			},
			{
				Name: "two namespace same dir",
				Objects: []ast.FileObject{
					fake.Namespace("namespaces/bar/ns-1.yaml"),
					fake.Namespace("namespaces/bar/ns-2.yaml"),
				},
				ShouldFail: true,
			},
			{
				Name: "two namespace different dir",
				Objects: []ast.FileObject{
					fake.Namespace("namespaces/bar/ns.yaml"),
					fake.Namespace("namespaces/foo/ns.yaml"),
				},
			},
		},
	}

	test.RunAll(t)
}
