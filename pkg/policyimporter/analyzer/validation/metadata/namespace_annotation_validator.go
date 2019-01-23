package metadata

import (
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
)

func init() {
	Register(NamespaceAnnotationValidatorFactory)
}

// NamespaceAnnotationValidatorFactory returns errors if a Namespace has the NamespaceSelector annotation.
var NamespaceAnnotationValidatorFactory = SyntaxValidatorFactory{
	fn: func(meta ResourceMeta) error {
		if meta.GroupVersionKind() == kinds.Namespace() {
			if _, found := meta.MetaObject().GetAnnotations()[v1alpha1.NamespaceSelectorAnnotationKey]; found {
				return vet.IllegalNamespaceAnnotationError{Resource: meta}
			}
		}
		return nil
	},
}
