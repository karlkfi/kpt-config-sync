package vet

import (
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/status"
)

// InvalidDirectoryNameErrorCode is the error code for InvalidDirectoryNameError
const InvalidDirectoryNameErrorCode = "1028"

func init() {
	status.AddExamples(InvalidDirectoryNameErrorCode, InvalidDirectoryNameError(cmpath.FromSlash("namespaces/a.b`c")))
}

var invalidDirectoryNameError = status.NewErrorBuilder(InvalidDirectoryNameErrorCode)

// InvalidDirectoryNameError represents an illegal usage of a reserved name.
func InvalidDirectoryNameError(dir cmpath.Path) status.Error {
	return invalidDirectoryNameError.WithPaths(dir).Errorf("Directory names must have fewer than 64 characters, consist of lower case alphanumeric characters or '-', and must "+
		"start and end with an alphanumeric character. Rename or remove the %q directory:", dir.Base())
}
