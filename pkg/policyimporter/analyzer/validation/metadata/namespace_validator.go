package metadata

import (
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	"github.com/google/nomos/pkg/policyimporter/analyzer/visitor"
)

// NewNamespaceValidator validates the value of metadata.namespace
func NewNamespaceValidator() *visitor.ValidatorVisitor {
	return visitor.NewAllObjectValidator(
		func(o ast.FileObject) error {
			if o.MetaObject().GetNamespace() != "" {
				return vet.IllegalMetadataNamespaceDeclarationError{Resource: &o}
			}
			return nil
		})
}
