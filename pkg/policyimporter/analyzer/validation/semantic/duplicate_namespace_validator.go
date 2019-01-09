package semantic

import (
	"path"

	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/veterrors"
	"github.com/google/nomos/pkg/util/multierror"
)

// DuplicateNamespaceValidator ensures no more than one Namespace is defined in a directory.
type DuplicateNamespaceValidator struct {
	Objects []ast.FileObject
}

// Validate adds errors to the errorBuilder if there are multiple Namespaces defined in directories.
func (v DuplicateNamespaceValidator) Validate(errorBuilder *multierror.Builder) {
	namespaces := make(map[string][]veterrors.ResourceID)

	for i, obj := range v.Objects {
		if obj.GroupVersionKind() == kinds.Namespace() {
			dir := path.Dir(obj.RelativeSlashPath())
			namespaces[dir] = append(namespaces[dir], &v.Objects[i])
		}
	}

	for _, namespaces := range namespaces {
		if len(namespaces) > 1 {
			errorBuilder.Add(veterrors.MultipleNamespacesError{Duplicates: namespaces})
		}
	}
}
