package metadata

import (
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/vet"
	"github.com/google/nomos/pkg/importer/analyzer/visitor"
	"github.com/google/nomos/pkg/status"
)

// NewNamespaceValidator validates the value of metadata.namespace
func NewNamespaceValidator() ast.Visitor {
	return visitor.NewAllObjectValidator(
		func(o ast.FileObject) status.MultiError {
			expected := o.Dir().Base()
			actual := o.GetNamespace()
			if actual != "" && actual != expected {
				return vet.IllegalMetadataNamespaceDeclarationError(&o, expected)
			}
			return nil
		})
}
