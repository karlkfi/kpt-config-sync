package syntax

import (
	"github.com/google/nomos/pkg/api/configmanagement"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/status"
)

// InvalidDirectoryNameErrorCode is the error code for InvalidDirectoryNameError
const InvalidDirectoryNameErrorCode = "1028"

var invalidDirectoryNameError = status.NewErrorBuilder(InvalidDirectoryNameErrorCode)

// ReservedDirectoryNameError represents an illegal usage of a reserved name.
func ReservedDirectoryNameError(dir cmpath.Relative) status.Error {
	// TODO(willbeason): Consider moving to Namespace validation instead.
	//  Strictly speaking, having a directory named "config-management-system" doesn't necessarily mean there are
	//  any resources declared in that Namespace. That would make this error message clearer.
	return invalidDirectoryNameError.
		Sprintf("%s repositories MUST NOT declare configs in the %s Namespace. Rename or remove the %q directory.",
			configmanagement.ProductName, configmanagement.ControllerNamespace, dir.Base()).
		BuildWithPaths(dir)
}

// InvalidDirectoryNameError represents an illegal usage of a reserved name.
func InvalidDirectoryNameError(dir cmpath.Relative) status.Error {
	return invalidDirectoryNameError.
		Sprintf(`Directory names MUST be valid Kubernetes Namespace names and must not be "config-management-system". Rename %q so that it:
1. has a length of 63 characters or fewer;
2. consists only of lowercase letters (a-z), digits (0-9), and hyphen '-';
3. begins and ends with a lowercase letter or digit`, dir.Base()).
		BuildWithPaths(dir)
}
