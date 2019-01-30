package filesystem

import (
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/validation/metadata"
	"github.com/google/nomos/pkg/policyimporter/analyzer/validation/semantic"
	"github.com/google/nomos/pkg/policyimporter/analyzer/validation/syntax"
	"github.com/google/nomos/pkg/policyimporter/filesystem/nomospath"
	"github.com/google/nomos/pkg/util/multierror"
)

// validateNamespaces validates all Resources in namespaces/ including Namespaces and Abstract
// Namespaces.
func validateNamespaces(
	objects []ast.FileObject,
	dirs []nomospath.Relative,
	errorBuilder *multierror.Builder) {
	metadata.Validate(toResourceMetas(objects), errorBuilder)

	syntax.DirectoryNameValidator.Validate(dirs, errorBuilder)
	syntax.DisallowSystemObjectsValidator.Validate(objects, errorBuilder)

	semantic.DuplicateDirectoryValidator{Dirs: dirs}.Validate(errorBuilder)
	semantic.DuplicateNamespaceValidator{Objects: objects}.Validate(errorBuilder)
}
