package metadata

import (
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/util/multierror"
)

// Validate validates metadata fields on the given Resources. These validations are
// Group/Version/Kind-independent.
func Validate(objects []ast.FileObject, errorBuilder *multierror.Builder) {
	AnnotationValidator.Validate(objects, errorBuilder)
	LabelValidator.Validate(objects, errorBuilder)
	MetadataNamespaceValidator.Validate(objects, errorBuilder)
	MetadataNameValidator.Validate(objects, errorBuilder)
	DuplicateNameValidator{Objects: objects}.Validate(errorBuilder)
}
