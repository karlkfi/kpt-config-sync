package metadata

import (
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	"github.com/google/nomos/pkg/policyimporter/analyzer/visitor"
	"github.com/google/nomos/pkg/status"
)

// NewNamespaceAnnotationValidator returns errors if a Namespace has the NamespaceSelector annotation.
func NewNamespaceAnnotationValidator() *visitor.ValidatorVisitor {
	return visitor.NewAllObjectValidator(
		func(o ast.FileObject) *status.MultiError {
			if o.GroupVersionKind() == kinds.Namespace() {
				if _, found := o.MetaObject().GetAnnotations()[v1.NamespaceSelectorAnnotationKey]; found {
					return status.From(vet.IllegalNamespaceAnnotationError{Resource: &o})
				}
			}
			return nil
		})
}
