package metadata

import (
	"testing"

	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast/asttesting"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	visitortesting "github.com/google/nomos/pkg/policyimporter/analyzer/visitor/testing"
	"github.com/google/nomos/pkg/policyimporter/filesystem/nomospath"
)

func fakeObjectWithNamespace(namespace string) ast.FileObject {
	object := asttesting.NewFakeObject(kinds.Role())
	object.SetNamespace(namespace)
	return ast.FileObject{Object: object, Relative: nomospath.NewFakeRelative("namespaces/role.yaml")}
}

func TestNamespaceValidator(t *testing.T) {
	test := visitortesting.ObjectValidatorTest{
		Validator: NewNamespaceValidator,
		ErrorCode: vet.IllegalMetadataNamespaceDeclarationErrorCode,
		TestCases: []visitortesting.ObjectValidatorTestCase{
			{
				Name:   "no metadata.namespace",
				Object: fakeObjectWithNamespace(""),
			},
			{
				Name:       "has metadata.namespace",
				Object:     fakeObjectWithNamespace("bar"),
				ShouldFail: true,
			},
		},
	}

	test.RunAll(t)
}
