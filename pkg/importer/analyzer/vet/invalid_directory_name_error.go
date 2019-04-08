package vet

import (
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/status"
)

// InvalidDirectoryNameErrorCode is the error code for InvalidDirectoryNameError
const InvalidDirectoryNameErrorCode = "1028"

func init() {
	status.Register(InvalidDirectoryNameErrorCode, InvalidDirectoryNameError{
		Dir: cmpath.FromSlash("namespaces/a.b`c"),
	})
}

// InvalidDirectoryNameError represents an illegal usage of a reserved name.
type InvalidDirectoryNameError struct {
	Dir cmpath.Path
}

var _ status.PathError = &InvalidDirectoryNameError{}

// Error implements error.
func (e InvalidDirectoryNameError) Error() string {
	return status.Format(e,
		"Directory names must have fewer than 64 characters, consist of lower case alphanumeric characters or '-', and must "+
			"start and end with an alphanumeric character. Rename or remove the %q directory:",
		e.Dir.Base())
}

// Code implements Error
func (e InvalidDirectoryNameError) Code() string { return InvalidDirectoryNameErrorCode }

// RelativePaths implements PathError
func (e InvalidDirectoryNameError) RelativePaths() []id.Path {
	return []id.Path{e.Dir}
}
