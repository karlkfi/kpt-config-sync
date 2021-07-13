package nonhierarchical

import (
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/metadata"
	"github.com/google/nomos/pkg/status"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ValidManagementAnnotation returns an Error if the user-specified management annotation is invalid.
func ValidManagementAnnotation(o ast.FileObject) status.Error {
	value, found := o.GetAnnotations()[metadata.ResourceManagementKey]
	if found && (value != metadata.ResourceManagementDisabled) {
		return IllegalManagementAnnotationError(&o, value)
	}
	return nil
}

// IllegalManagementAnnotationErrorCode is the error code for IllegalManagementAnnotationError.
const IllegalManagementAnnotationErrorCode = "1005"

var illegalManagementAnnotationError = status.NewErrorBuilder(IllegalManagementAnnotationErrorCode)

// IllegalManagementAnnotationError represents an illegal management annotation value.
// Error implements error.
func IllegalManagementAnnotationError(resource client.Object, value string) status.Error {
	return illegalManagementAnnotationError.
		Sprintf("Config has invalid management annotation %s=%s. If set, the value must be %q.",
			metadata.ResourceManagementKey, value, metadata.ResourceManagementDisabled).
		BuildWithResources(resource)
}
