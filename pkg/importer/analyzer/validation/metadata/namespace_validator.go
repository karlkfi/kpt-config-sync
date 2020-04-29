package metadata

import (
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/visitor"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/status"
)

// NewMetadataNamespaceDirectoryValidator validates the value of metadata.namespace
func NewMetadataNamespaceDirectoryValidator() ast.Visitor {
	return visitor.NewAllObjectValidator(
		func(o ast.FileObject) status.MultiError {
			expected := o.Dir().Base()
			actual := o.GetNamespace()
			if actual != "" && actual != expected {
				return IllegalMetadataNamespaceDeclarationError(&o, expected)
			}
			return nil
		})
}

// IllegalMetadataNamespaceDeclarationErrorCode is the error code for IllegalNamespaceDeclarationError
const IllegalMetadataNamespaceDeclarationErrorCode = "1009"

var illegalMetadataNamespaceDeclarationError = status.NewErrorBuilder(IllegalMetadataNamespaceDeclarationErrorCode)

// IllegalMetadataNamespaceDeclarationError represents illegally declaring metadata.namespace
func IllegalMetadataNamespaceDeclarationError(resource id.Resource, expectedNamespace string) status.Error {
	return illegalMetadataNamespaceDeclarationError.
		Sprintf("A config MUST either declare a `namespace` field exactly matching the directory "+
			"containing the config, %q, or leave the field blank:", expectedNamespace).
		BuildWithResources(resource)
}
