package metadata

import (
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
)

func init() {
	Register(NamespaceValidatorFactory)
}

// NamespaceValidatorFactory validates the value of metadata.namespace
var NamespaceValidatorFactory = SyntaxValidatorFactory{
	fn: func(meta ResourceMeta) error {
		if meta.MetaObject().GetNamespace() != "" {
			return vet.IllegalMetadataNamespaceDeclarationError{Resource: meta}
		}
		return nil
	},
}
