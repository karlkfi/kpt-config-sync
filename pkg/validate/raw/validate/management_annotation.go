package validate

import (
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical"
	"github.com/google/nomos/pkg/metadata"
	"github.com/google/nomos/pkg/status"
)

// ManagementAnnotation returns an Error if the user-specified management annotation is invalid.
func ManagementAnnotation(obj ast.FileObject) status.Error {
	value, found := obj.GetAnnotations()[metadata.ResourceManagementKey]
	if found && (value != metadata.ResourceManagementDisabled) {
		return nonhierarchical.IllegalManagementAnnotationError(obj, value)
	}
	return nil
}
