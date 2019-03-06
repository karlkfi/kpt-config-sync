package metadata

import (
	"testing"

	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	"github.com/google/nomos/pkg/testing/asttest"
	"github.com/google/nomos/pkg/testing/object"
)

func TestNamespaceValidator(t *testing.T) {
	test := asttest.Validator(
		NewNamespaceValidator,
		vet.IllegalMetadataNamespaceDeclarationErrorCode,
		asttest.Pass("no metadata.namespace",
			object.Build(kinds.Role(), object.Path("namespaces/foo/role.yaml"), object.Namespace("")),
		),

		asttest.Fail("wrong metadata.namespace",
			object.Build(kinds.Role(), object.Path("namespaces/foo/role.yaml"), object.Namespace("bar")),
		),

		asttest.Pass("correct metadata.namespace",
			object.Build(kinds.Role(), object.Path("namespaces/foo/role.yaml"), object.Namespace("foo")),
		),
	)

	test.RunAll(t)
}
