package metadata

import (
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/vet"
	"github.com/google/nomos/pkg/importer/analyzer/visitor"
	"github.com/google/nomos/pkg/status"
)

// NewLabelValidator validates the labels declared in metadata
func NewLabelValidator() ast.Visitor {
	return visitor.NewAllObjectValidator(
		func(o ast.FileObject) status.MultiError {
			var errors []string
			for l := range o.MetaObject().GetLabels() {
				if hasConfigManagementPrefix(l) {
					errors = append(errors, l)
				}
			}
			if errors != nil {
				return status.From(vet.IllegalLabelDefinitionError{Resource: &o, Labels: errors})
			}
			return nil
		})
}
