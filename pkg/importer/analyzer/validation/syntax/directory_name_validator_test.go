package syntax

import (
	"testing"

	"github.com/google/nomos/pkg/api/configmanagement"
	"github.com/google/nomos/pkg/testing/asttest"
	"github.com/google/nomos/pkg/testing/fake"
)

func TestDirectoryNameValidator_Pass(t *testing.T) {
	asttest.Validator(t, NewDirectoryNameValidator,
		InvalidDirectoryNameErrorCode,
		asttest.Pass("valid name", fake.Namespace("namespaces/foo")),
		asttest.Fail("invalid name", fake.Namespace("namespaces/...")),
		asttest.Fail("controller namespace", fake.Namespace("namespaces/"+configmanagement.ControllerNamespace)))
}
