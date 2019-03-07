package vet

import (
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1/repo"
	"github.com/google/nomos/pkg/status"
)

// MissingRepoErrorCode is the error code for MissingRepoError
const MissingRepoErrorCode = "1017"

func init() {
	register(MissingRepoErrorCode, nil, "")
}

// MissingRepoError reports that there is no Repo definition in system/
type MissingRepoError struct{}

// Error implements error
func (e MissingRepoError) Error() string {
	return status.Format(e,
		"%s/ directory must declare a Repo Resource.", repo.SystemDir)
}

// Code implements Error
func (e MissingRepoError) Code() string { return MissingRepoErrorCode }
