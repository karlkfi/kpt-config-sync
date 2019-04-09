package vet

import (
	"github.com/google/nomos/pkg/api/configmanagement/v1/repo"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/status"
)

// IllegalKindInSystemErrorCode is the error code for IllegalKindInSystemError
const IllegalKindInSystemErrorCode = "1024"

func init() {
	status.Register(IllegalKindInSystemErrorCode, IllegalKindInSystemError{
		Resource: role(),
	})
}

// IllegalKindInSystemError reports that an object has been illegally defined in system/
type IllegalKindInSystemError struct {
	id.Resource
}

var _ status.ResourceError = &IllegalKindInSystemError{}

// Error implements error
func (e IllegalKindInSystemError) Error() string {
	return status.Format(e,
		"Configs of this Kind may not be declared in the `%s/` directory of the repo:",
		repo.SystemDir)
}

// Code implements Error
func (e IllegalKindInSystemError) Code() string {
	return IllegalKindInSystemErrorCode
}

// Resources implements ResourceError
func (e IllegalKindInSystemError) Resources() []id.Resource {
	return []id.Resource{e.Resource}
}
