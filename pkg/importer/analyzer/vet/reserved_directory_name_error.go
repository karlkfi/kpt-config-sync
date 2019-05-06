package vet

import (
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/status"
)

// ReservedDirectoryNameErrorCode is the error code for ReservedDirectoryNameError
const ReservedDirectoryNameErrorCode = "1001"

func init() {
	status.AddExamples(ReservedDirectoryNameErrorCode, ReservedDirectoryNameError(cmpath.FromSlash("namespaces/reserved")))
}

var reservedDirectoryNameError = status.NewErrorBuilder(ReservedDirectoryNameErrorCode)

// ReservedDirectoryNameError represents an illegal usage of a reserved name.
func ReservedDirectoryNameError(dir cmpath.Path) status.Error {
	return reservedDirectoryNameError.Errorf("Directories MUST NOT have reserved namespace names. Rename or remove %q:\n\n" +
		dir.Base())
}
