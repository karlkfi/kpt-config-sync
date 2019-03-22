package vet

import (
	"github.com/google/nomos/pkg/api/configmanagement/v1/repo"
	"github.com/google/nomos/pkg/status"
)

// MissingRepoErrorCode is the error code for MissingRepoError
const MissingRepoErrorCode = "1017"

func init() {
	status.Register(MissingRepoErrorCode, MissingRepoError{})
}

// MissingRepoError reports that there is no Repo definition in system/
type MissingRepoError struct{}

var _ status.PathError = &MissingRepoError{}

// Error implements error
func (e MissingRepoError) Error() string {
	return status.Format(e,
		"%s/ directory must declare a Repo Resource.", repo.SystemDir)
}

// Code implements Error
func (e MissingRepoError) Code() string { return MissingRepoErrorCode }

// RelativePaths implements PathError
func (e MissingRepoError) RelativePaths() []string {
	return []string{repo.SystemDir}
}
