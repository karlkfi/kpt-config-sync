package semantic

import (
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/veterrors"
	"github.com/google/nomos/pkg/policyimporter/filesystem/nomospath"
	"github.com/google/nomos/pkg/policyimporter/id"
	"github.com/google/nomos/pkg/util/multierror"
)

// DuplicateNamespaceValidator ensures no more than one Namespace is defined in a directory.
type DuplicateNamespaceValidator struct {
	Objects []ast.FileObject
}

// Validate adds errors to the errorBuilder if there are multiple Namespaces defined in directories.
func (v DuplicateNamespaceValidator) Validate(errorBuilder *multierror.Builder) {
	namespaces := make(map[nomospath.Relative][]id.Resource)

	for i, obj := range v.Objects {
		if obj.GroupVersionKind() == kinds.Namespace() {
			dir := obj.Dir()
			namespaces[dir] = append(namespaces[dir], &v.Objects[i])
		}
	}

	for _, namespaces := range namespaces {
		if len(namespaces) > 1 {
			errorBuilder.Add(veterrors.MultipleNamespacesError{Duplicates: namespaces})
		}
	}
}
