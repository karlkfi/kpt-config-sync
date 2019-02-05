package metadata

import (
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	"github.com/google/nomos/pkg/policyimporter/analyzer/visitor"
)

// NewAnnotationValidator validates the annotations of every object.
func NewAnnotationValidator() *visitor.ValidatorVisitor {
	return visitor.NewAllObjectValidator(
		func(o ast.FileObject) error {
			var errors []string
			for a := range o.MetaObject().GetAnnotations() {
				if !v1alpha1.IsInputAnnotation(a) && v1alpha1.HasNomosPrefix(a) {
					errors = append(errors, a)
				}
			}
			if errors != nil {
				return vet.IllegalAnnotationDefinitionError{Resource: &o, Annotations: errors}
			}
			return nil
		})
}
