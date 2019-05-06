package vet

import (
	"github.com/google/nomos/pkg/api/configmanagement/v1/repo"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/status"
)

// MissingRepoErrorCode is the error code for MissingRepoError
const MissingRepoErrorCode = "1017"

func init() {
	status.AddExamples(MissingRepoErrorCode, MissingRepoError())
}

var missingRepoError = status.NewErrorBuilder(MissingRepoErrorCode)

// MissingRepoError reports that there is no Repo definition in system/
func MissingRepoError() status.Error {
	return missingRepoError.WithPaths(cmpath.FromSlash(repo.SystemDir)).Errorf("The %s/ directory must declare a Repo Resource.", repo.SystemDir)
}
