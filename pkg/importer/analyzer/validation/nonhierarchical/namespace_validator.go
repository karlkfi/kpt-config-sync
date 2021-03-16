package nonhierarchical

import (
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/status"
)

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
