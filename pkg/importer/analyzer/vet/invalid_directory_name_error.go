package vet

import (
	"strings"

	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/importer/id"
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
	return invalidDirectoryNameError.WithPaths(dir).Errorf(
		"Directory names must have fewer than 64 characters, consist of lower case alphanumeric characters or '-', and must "+
			"start and end with an alphanumeric character. Rename or remove the %q directory:", dir.Base())
}

// InvalidNamespaceError reports using an illegal Namespace.
func InvalidNamespaceError(o id.Resource, errs []string) status.Error {
	return invalidDirectoryNameError.WithResources(o).Errorf(
		"metadata.namespace is invalid:\n\n%s\n", strings.Join(errs, "\n"))
}
