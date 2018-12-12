package syntax

import (
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
)

// MetadataNamespaceValidator validates the value of metadata.namespace
var MetadataNamespaceValidator = &FileObjectValidator{
	validate: func(fileObject ast.FileObject) error {
		if fileObject.ToMeta().GetNamespace() != "" {
			return vet.IllegalMetadataNamespaceDeclarationError{Object: fileObject}
		}
		return nil
	},
}
