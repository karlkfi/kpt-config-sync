package metadata

import (
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/ast/node"
	"github.com/google/nomos/pkg/importer/analyzer/visitor"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
)

// NewNamespaceAnnotationValidator returns errors if a Namespace has the NamespaceSelector annotation.
func NewNamespaceAnnotationValidator() *visitor.ValidatorVisitor {
	return visitor.NewAllObjectValidator(
		func(o ast.FileObject) status.MultiError {
			if o.GroupVersionKind() == kinds.Namespace() {
				if _, found := o.GetAnnotations()[v1.NamespaceSelectorAnnotationKey]; found {
					return IllegalNamespaceAnnotationError(&o)
				}
			}
			return nil
		})
}

// IllegalNamespaceAnnotationErrorCode is the error code for IllegalNamespaceAnnotationError
const IllegalNamespaceAnnotationErrorCode = "1004"

var illegalNamespaceAnnotationError = status.NewErrorBuilder(IllegalNamespaceAnnotationErrorCode)

// IllegalNamespaceAnnotationError represents an illegal usage of the namespace selector annotation.
func IllegalNamespaceAnnotationError(resource id.Resource) status.Error {
	return illegalNamespaceAnnotationError.
		Sprintf("A %s MUST NOT use the annotation %s. "+
			"Remove metadata.annotations.%[2]s from:",
			node.Namespace, v1.NamespaceSelectorAnnotationKey).
		BuildWithResources(resource)
}
