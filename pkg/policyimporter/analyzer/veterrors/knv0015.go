package veterrors

import "github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1/repo"

// MissingDirectoryErrorCode is the error code for MissingDirectoryError
const MissingDirectoryErrorCode = "1015"

func init() {
	register(MissingDirectoryErrorCode, nil, "")
}

// MissingDirectoryError reports that a required directory is missing.
type MissingDirectoryError struct{}

// Error implements error.
func (e MissingDirectoryError) Error() string {
	return format(e,
		"Required %s/ directory is missing.", repo.SystemDir)
}

// Code implements Error
func (e MissingDirectoryError) Code() string { return MissingDirectoryErrorCode }
