package veterrors

import "github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1/repo"

// MissingDirectoryError reports that a required directory is missing.
type MissingDirectoryError struct{}

// Error implements error.
func (e MissingDirectoryError) Error() string {
	return format(e,
		"Required %s/ directory is missing.", repo.SystemDir)
}

// Code implements Error
func (e MissingDirectoryError) Code() string { return MissingDirectoryErrorCode }
