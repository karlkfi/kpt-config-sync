package nonhierarchical

import (
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/status"
)

// ManagementAnnotationValidator ensures the passed object either has no Managment annotation, or
//  declares a valid one.
var ManagementAnnotationValidator = PerObjectValidator(ValidManagementAnnotation)

// ValidManagementAnnotation returns an Error if the user-specified Managment annotation is invalid.
func ValidManagementAnnotation(o ast.FileObject) status.Error {
	value, found := o.GetAnnotations()[v1.ResourceManagementKey]
	if found && (value != v1.ResourceManagementDisabled) {
		return IllegalManagementAnnotationError(&o, value)
	}
	return nil
}

// IllegalManagementAnnotationErrorCode is the error code for IllegalManagementAnnotationError.
const IllegalManagementAnnotationErrorCode = "1005"

var illegalManagementAnnotationError = status.NewErrorBuilder(IllegalManagementAnnotationErrorCode)

// IllegalManagementAnnotationError represents an illegal management annotation value.
// Error implements error.
func IllegalManagementAnnotationError(resource id.Resource, value string) status.Error {
	return illegalManagementAnnotationError.
		Sprintf("Config has invalid management annotation %s=%s. Must be %s or unset.",
			v1.ResourceManagementKey, value, v1.ResourceManagementDisabled).
		BuildWithResources(resource)
}
