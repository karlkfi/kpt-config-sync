package metadata

import (
	"testing"

	"github.com/google/nomos/pkg/importer/analyzer/vet"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/object"
	"github.com/google/nomos/pkg/testing/asttest"
	"github.com/google/nomos/pkg/testing/fake"
)

func TestNamespaceValidator(t *testing.T) {
	test := asttest.Validator(
		NewNamespaceValidator,
		vet.IllegalMetadataNamespaceDeclarationErrorCode,
		asttest.Pass("no metadata.namespace",
			fake.Build(kinds.Role(), object.Path("namespaces/foo/role.yaml"), object.Namespace("")),
		),

		asttest.Fail("wrong metadata.namespace",
			fake.Build(kinds.Role(), object.Path("namespaces/foo/role.yaml"), object.Namespace("bar")),
		),

		asttest.Pass("correct metadata.namespace",
			fake.Build(kinds.Role(), object.Path("namespaces/foo/role.yaml"), object.Namespace("foo")),
		),
	)

	test.RunAll(t)
}
