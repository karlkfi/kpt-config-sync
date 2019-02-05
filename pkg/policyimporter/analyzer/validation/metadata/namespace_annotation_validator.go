package metadata

import (
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	"github.com/google/nomos/pkg/policyimporter/analyzer/visitor"
)

// NewNamespaceAnnotationValidator returns errors if a Namespace has the NamespaceSelector annotation.
func NewNamespaceAnnotationValidator() *visitor.ValidatorVisitor {
	return visitor.NewAllObjectValidator(
		func(o ast.FileObject) error {
			if o.GroupVersionKind() == kinds.Namespace() {
				if _, found := o.MetaObject().GetAnnotations()[v1alpha1.NamespaceSelectorAnnotationKey]; found {
					return vet.IllegalNamespaceAnnotationError{Resource: &o}
				}
			}
			return nil
		})
}
