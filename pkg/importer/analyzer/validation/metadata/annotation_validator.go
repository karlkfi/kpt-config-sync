package metadata

import (
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/vet"
	"github.com/google/nomos/pkg/importer/analyzer/visitor"
	"github.com/google/nomos/pkg/status"
)

// NewAnnotationValidator validates the annotations of every object.
func NewAnnotationValidator() ast.Visitor {
	return visitor.NewAllObjectValidator(
		func(o ast.FileObject) *status.MultiError {
			var errors []string
			for a := range o.MetaObject().GetAnnotations() {
				if !v1.IsInputAnnotation(a) && v1.HasConfigManagementPrefix(a) {
					errors = append(errors, a)
				}
			}
			if errors != nil {
				return status.From(vet.IllegalAnnotationDefinitionError{Resource: &o, Annotations: errors})
			}
			return nil
		})
}

// NewManagedAnnotationValidator validates the value of the management annotation label.
func NewManagedAnnotationValidator() ast.Visitor {
	return visitor.NewAllObjectValidator(
		func(o ast.FileObject) *status.MultiError {
			value, found := o.MetaObject().GetAnnotations()[v1.ResourceManagementKey]
			if found && (value != v1.ResourceManagementDisabled) {
				return status.From(vet.IllegalManagementAnnotationError{
					Resource: &o,
					Value:    value,
				})
			}
			return nil
		})
}
