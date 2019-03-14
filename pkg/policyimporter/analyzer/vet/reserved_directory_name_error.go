package vet

import (
	"github.com/google/nomos/pkg/policyimporter/filesystem/nomospath"
	"github.com/google/nomos/pkg/status"
)

// ReservedDirectoryNameErrorCode is the error code for ReservedDirectoryNameError
const ReservedDirectoryNameErrorCode = "1001"

func init() {
	register(ReservedDirectoryNameErrorCode)
}

// ReservedDirectoryNameError represents an illegal usage of a reserved name.
type ReservedDirectoryNameError struct {
	Dir nomospath.Path
}

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
