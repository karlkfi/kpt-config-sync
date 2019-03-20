package vet

import (
	"github.com/google/nomos/pkg/policyimporter/filesystem/cmpath"
	"github.com/google/nomos/pkg/status"
)

// ReservedDirectoryNameErrorCode is the error code for ReservedDirectoryNameError
const ReservedDirectoryNameErrorCode = "1001"

func init() {
	status.Register(ReservedDirectoryNameErrorCode, ReservedDirectoryNameError{})
}

// ReservedDirectoryNameError represents an illegal usage of a reserved name.
type ReservedDirectoryNameError struct {
	Dir cmpath.Path
}

var _ status.PathError = &ReservedDirectoryNameError{}

// Error implements error.
func (e ReservedDirectoryNameError) Error() string {
	return status.Format(e,
		"Directories MUST NOT have reserved namespace names. Rename or remove directory:\n\n"+
			"path: %[1]s\n"+
			"name: %[2]s",
		e.Dir.SlashPath(), e.Dir.Base())
}

// Code implements Error
func (e ReservedDirectoryNameError) Code() string { return ReservedDirectoryNameErrorCode }

// RelativePaths implements PathError
func (e ReservedDirectoryNameError) RelativePaths() []string {
	return []string{e.Dir.SlashPath()}
}
