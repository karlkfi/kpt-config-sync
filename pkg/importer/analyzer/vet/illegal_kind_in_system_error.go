package vet

import (
	"github.com/google/nomos/pkg/api/configmanagement/v1/repo"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/status"
)

// IllegalKindInSystemErrorCode is the error code for IllegalKindInSystemError
const IllegalKindInSystemErrorCode = "1024"

func init() {
	status.AddExamples(IllegalKindInSystemErrorCode, IllegalKindInSystemError(
		role(),
	))
}

var illegalKindInSystemError = status.NewErrorBuilder(IllegalKindInSystemErrorCode)

// IllegalKindInSystemError reports that an object has been illegally defined in system/
func IllegalKindInSystemError(resource id.Resource) status.Error {
	return illegalKindInSystemError.WithResources(resource).Errorf(
		"Configs of this Kind may not be declared in the `%s/` directory of the repo:",
		repo.SystemDir)
}
