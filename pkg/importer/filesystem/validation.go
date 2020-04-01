package filesystem

import (
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical"
	"github.com/google/nomos/pkg/status"
)

// standardValidation is validation run on all KubernetesObjects, regardless of whether the source was a hierarchical or
// nonhierarchical repository.
func standardValidation(fileObjects []ast.FileObject) status.MultiError {
	var validators = []nonhierarchical.Validator{
		nonhierarchical.DisallowedFieldsValidator,
		nonhierarchical.DuplicateNameValidator,
		nonhierarchical.IllegalNamespaceValidator,
		nonhierarchical.NameValidator,
		nonhierarchical.NamespaceValidator,
		nonhierarchical.ManagementAnnotationValidator,
		nonhierarchical.CRDNameValidator,
		nonhierarchical.IllegalCRDValidator,
		nonhierarchical.ManagedNamespaceValidator,
	}

	var errs status.MultiError
	for _, v := range validators {
		errs = status.Append(errs, v.Validate(fileObjects))
	}
	return errs
}
