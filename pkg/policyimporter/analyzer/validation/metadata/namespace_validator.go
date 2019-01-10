package metadata

import (
	"github.com/google/nomos/pkg/policyimporter/analyzer/veterrors"
)

// NamespaceValidatorFactory validates the value of metadata.namespace
var NamespaceValidatorFactory = ValidatorFactory{
	fn: func(meta ResourceMeta) error {
		if meta.MetaObject().GetNamespace() != "" {
			return veterrors.IllegalMetadataNamespaceDeclarationError{Resource: meta}
		}
		return nil
	},
}
