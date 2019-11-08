package metadata

import (
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/vet"
	"github.com/google/nomos/pkg/importer/analyzer/visitor"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
)

// NewNamespaceAnnotationValidator returns errors if a Namespace has the NamespaceSelector annotation.
func NewNamespaceAnnotationValidator() *visitor.ValidatorVisitor {
	return visitor.NewAllObjectValidator(
		func(o ast.FileObject) status.MultiError {
			if o.GroupVersionKind() == kinds.Namespace() {
				if _, found := o.GetAnnotations()[v1.NamespaceSelectorAnnotationKey]; found {
					return vet.IllegalNamespaceAnnotationError(&o)
				}
			}
			return nil
		})
}
