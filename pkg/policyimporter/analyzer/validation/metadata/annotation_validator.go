package metadata

import (
	v1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	"github.com/google/nomos/pkg/policyimporter/analyzer/visitor"
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
