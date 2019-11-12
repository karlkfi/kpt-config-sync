package metadata

import (
	"testing"

	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/testing/asttest"
	"github.com/google/nomos/pkg/testing/fake"
)

func TestNamespaceValidator(t *testing.T) {
	test := asttest.Validator(
		NewNamespaceValidator,
		IllegalMetadataNamespaceDeclarationErrorCode,
		asttest.Pass("no metadata.namespace",
			fake.RoleAtPath("namespaces/foo/role.yaml", core.Namespace("")),
		),

		asttest.Fail("wrong metadata.namespace",
			fake.RoleAtPath("namespaces/foo/role.yaml", core.Namespace("bar")),
		),

		asttest.Pass("correct metadata.namespace",
			fake.RoleAtPath("namespaces/foo/role.yaml", core.Namespace("foo")),
		),
	)

	test.RunAll(t)
}
