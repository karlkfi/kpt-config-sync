package metadata

import (
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/validation/syntax"
	"github.com/google/nomos/pkg/policyimporter/analyzer/veterrors"
)

// MetadataNamespaceValidator validates the value of metadata.namespace
var MetadataNamespaceValidator = &syntax.FileObjectValidator{
	ValidateFn: func(fileObject ast.FileObject) error {
		if fileObject.ToMeta().GetNamespace() != "" {
			return veterrors.IllegalMetadataNamespaceDeclarationError{ResourceID: &fileObject}
		}
		return nil
	},
}
