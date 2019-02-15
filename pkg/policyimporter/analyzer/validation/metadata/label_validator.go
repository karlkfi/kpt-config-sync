package metadata

import (
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	"github.com/google/nomos/pkg/policyimporter/analyzer/visitor"
)

// NewLabelValidator validates the labels declared in metadata
func NewLabelValidator() ast.Visitor {
	return visitor.NewAllObjectValidator(
		func(o ast.FileObject) error {
			var errors []string
			for l := range o.MetaObject().GetLabels() {
				if v1alpha1.HasNomosPrefix(l) {
					errors = append(errors, l)
				}
			}
			if errors != nil {
				return vet.IllegalLabelDefinitionError{Resource: &o, Labels: errors}
			}
			return nil
		})
}
