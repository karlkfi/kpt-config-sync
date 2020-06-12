package nonhierarchical

import (
	"github.com/google/nomos/pkg/importer/id"
	"k8s.io/apimachinery/pkg/util/validation"

	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/status"
)

// NamespaceValidator validates the metadata.namespace field on resources.
var NamespaceValidator = PerObjectValidator(validNamespace)

func validNamespace(o ast.FileObject) status.Error {
	if o.GetNamespace() == "" {
		return nil
	}

	// Do note that IsDNS1123Label is misleading. It refers to RFC 1123, and additionally forbids
	// capital letters.
	errs := validation.IsDNS1123Label(o.GetNamespace())
	if errs != nil {
		return InvalidNamespaceError(&o)
	}
	return nil
}

// InvalidDirectoryNameErrorCode is the error code for InvalidDirectoryNameError
const InvalidDirectoryNameErrorCode = "1055"

var invalidDirectoryNameError = status.NewErrorBuilder(InvalidDirectoryNameErrorCode)

// InvalidNamespaceError reports using an illegal Namespace.
func InvalidNamespaceError(o id.Resource) status.Error {
	return invalidDirectoryNameError.
		Sprintf(`metadata.namespace MUST be valid Kubernetes Namespace names. Rename %q so that it:

1. has a length of 63 characters or fewer;
2. consists only of lowercase letters (a-z), digits (0-9), and hyphen '-'; and
3. begins and ends with a lowercase letter or digit.
`, o.GetNamespace()).
		BuildWithResources(o)
}
