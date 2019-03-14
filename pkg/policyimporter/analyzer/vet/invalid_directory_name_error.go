package vet

import (
	"github.com/google/nomos/pkg/policyimporter/filesystem/nomospath"
	"github.com/google/nomos/pkg/status"
)

// InvalidDirectoryNameErrorCode is the error code for InvalidDirectoryNameError
const InvalidDirectoryNameErrorCode = "1028"

func init() {
	register(InvalidDirectoryNameErrorCode)
}

// InvalidDirectoryNameError represents an illegal usage of a reserved name.
type InvalidDirectoryNameError struct {
	Dir nomospath.Path
}

var _ status.PathError = &InvalidDirectoryNameError{}

// Error implements error.
func (e InvalidDirectoryNameError) Error() string {
	return status.Format(e,
		"Directories MUST be a valid RFC1123 DNS label. Rename or remove directory:\n\n"+
			"path: %[1]s\n"+
			"name: %[2]s",
		e.Dir.SlashPath(), e.Dir.Base())
}

// Code implements Error
func (e InvalidDirectoryNameError) Code() string { return InvalidDirectoryNameErrorCode }

// RelativePaths implements PathError
func (e InvalidDirectoryNameError) RelativePaths() []string {
	return []string{e.Dir.SlashPath()}
}
