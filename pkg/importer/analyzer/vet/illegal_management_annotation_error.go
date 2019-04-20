package vet

import (
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/status"
)

// IllegalManagementAnnotationErrorCode is the error code for IllegalManagementAnnotationError.
const IllegalManagementAnnotationErrorCode = "1005"

func init() {
	status.Register(IllegalManagementAnnotationErrorCode, IllegalManagementAnnotationError{
		Resource: role(),
		Value:    "invalid",
	})
}

// IllegalManagementAnnotationError represents an illegal management annotation value.
type IllegalManagementAnnotationError struct {
	id.Resource
	Value string
}

var _ status.ResourceError = IllegalManagementAnnotationError{}

// Error implements error.
func (e IllegalManagementAnnotationError) Error() string {
	return status.Format(e, "Config has invalid management annotation %s=%s. Must be %s or unset.",
		v1.ResourceManagementKey, e.Value, v1.ResourceManagementDisabled)
}

// Code implements Error.
func (e IllegalManagementAnnotationError) Code() string {
	return IllegalManagementAnnotationErrorCode
}

// Resources implements ResourceError.
func (e IllegalManagementAnnotationError) Resources() []id.Resource {
	return []id.Resource{e.Resource}
}

// ToCME implements ToCMEr.
func (e IllegalManagementAnnotationError) ToCME() v1.ConfigManagementError {
	return status.FromResourceError(e)
}
