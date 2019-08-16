package metadata

import (
	"strings"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/vet"
	"github.com/google/nomos/pkg/importer/analyzer/visitor"
	"github.com/google/nomos/pkg/status"
)

// inputAnnotations is a map of annotations that are valid to exist on objects when imported from
// the filesystem.
var inputAnnotations = map[string]bool{
	v1.NamespaceSelectorAnnotationKey: true,
	v1.ClusterSelectorAnnotationKey:   true,
	v1.ResourceManagementKey:          true,
}

// isInputAnnotation returns true if the annotation is a Nomos input annotation.
func isInputAnnotation(s string) bool {
	return inputAnnotations[s]
}

// hasConfigManagementPrefix returns true if the string begins with the Nomos annotation prefix.
func hasConfigManagementPrefix(s string) bool {
	return strings.HasPrefix(s, v1.ConfigManagementPrefix)
}

// NewAnnotationValidator validates the annotations of every object.
func NewAnnotationValidator() ast.Visitor {
	return visitor.NewAllObjectValidator(
		func(o ast.FileObject) status.MultiError {
			var errors []string
			for a := range o.MetaObject().GetAnnotations() {
				if !isInputAnnotation(a) && hasConfigManagementPrefix(a) {
					errors = append(errors, a)
				}
			}
			if errors != nil {
				return status.From(vet.IllegalAnnotationDefinitionError(&o, errors))
			}
			return nil
		})
}

// NewManagedAnnotationValidator validates the value of the management annotation label.
func NewManagedAnnotationValidator() ast.Visitor {
	return visitor.NewAllObjectValidator(
		func(o ast.FileObject) status.MultiError {
			return status.From(ValidManagementAnnotation(o))
		})
}

// ValidManagementAnnotation returns an Error if the user-specified Managment annotation is invalid.
func ValidManagementAnnotation(o ast.FileObject) status.Error {
	value, found := o.MetaObject().GetAnnotations()[v1.ResourceManagementKey]
	if found && (value != v1.ResourceManagementDisabled) {
		return vet.IllegalManagementAnnotationError(&o, value)
	}
	return nil
}
