package metadata

import (
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	"github.com/google/nomos/pkg/policyimporter/analyzer/visitor"
	"github.com/google/nomos/pkg/status"
)

// NewNamespaceValidator validates the value of metadata.namespace
func NewNamespaceValidator() ast.Visitor {
	return visitor.NewAllObjectValidator(
		func(o ast.FileObject) *status.MultiError {
			expected := o.Dir().Base()
			actual := o.MetaObject().GetNamespace()
			if actual != "" && actual != expected {
				return status.From(vet.IllegalMetadataNamespaceDeclarationError{Resource: &o})
			}
			return nil
		})
}
