package vet

import (
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/status"
)

// IllegalManagementAnnotationErrorCode is the error code for IllegalManagementAnnotationError.
const IllegalManagementAnnotationErrorCode = "1005"

func init() {
	status.AddExamples(IllegalManagementAnnotationErrorCode, IllegalManagementAnnotationError(
		role(),
		"invalid",
	))
}

var illegalManagementAnnotationError = status.NewErrorBuilder(IllegalManagementAnnotationErrorCode)

// IllegalManagementAnnotationError represents an illegal management annotation value.
// Error implements error.
func IllegalManagementAnnotationError(resource id.Resource, value string) status.Error {
	return illegalManagementAnnotationError.WithResources(resource).Errorf("Config has invalid management annotation %s=%s. Must be %s or unset.",
		v1.ResourceManagementKey, value, v1.ResourceManagementDisabled)
}
