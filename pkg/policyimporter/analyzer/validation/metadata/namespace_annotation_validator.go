package metadata

import (
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/policyimporter/analyzer/veterrors"
)

// NamespaceAnnotationValidatorFactory returns errors if a Namespace has the NamespaceSelector annotation.
var NamespaceAnnotationValidatorFactory = ValidatorFactory{
	fn: func(meta ResourceMeta) error {
		if meta.GroupVersionKind() == kinds.Namespace() {
			if _, found := meta.MetaObject().GetAnnotations()[v1alpha1.NamespaceSelectorAnnotationKey]; found {
				return veterrors.IllegalNamespaceAnnotationError{Resource: meta}
			}
		}
		return nil
	},
}
