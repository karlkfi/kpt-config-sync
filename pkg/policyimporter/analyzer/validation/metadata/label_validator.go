package metadata

import (
	v1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	"github.com/google/nomos/pkg/policyimporter/analyzer/visitor"
	"github.com/google/nomos/pkg/status"
)

// NewLabelValidator validates the labels declared in metadata
func NewLabelValidator() ast.Visitor {
	return visitor.NewAllObjectValidator(
		func(o ast.FileObject) *status.MultiError {
			var errors []string
			for l := range o.MetaObject().GetLabels() {
				if v1.HasNomosPrefix(l) {
					errors = append(errors, l)
				}
			}
			if errors != nil {
				return status.From(vet.IllegalLabelDefinitionError{Resource: &o, Labels: errors})
			}
			return nil
		})
}
