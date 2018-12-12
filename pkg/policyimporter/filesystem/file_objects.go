package filesystem

import (
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/validation/semantic"
	"github.com/google/nomos/pkg/policyimporter/analyzer/validation/syntax"
	"github.com/google/nomos/pkg/util/multierror"
)

// validation run on all objects
func validateObjects(objects []ast.FileObject, errorBuilder *multierror.Builder) {
	syntax.AnnotationValidator.Validate(objects, errorBuilder)
	syntax.LabelValidator.Validate(objects, errorBuilder)
	syntax.MetadataNamespaceValidator.Validate(objects, errorBuilder)
	syntax.MetadataNameValidator.Validate(objects, errorBuilder)

	semantic.DuplicateNameValidator{Objects: objects}.Validate(errorBuilder)
}

func toSources(infos []ast.FileObject) []string {
	result := make([]string, len(infos))
	for i, info := range infos {
		result[i] = info.Source
	}
	return result
}
