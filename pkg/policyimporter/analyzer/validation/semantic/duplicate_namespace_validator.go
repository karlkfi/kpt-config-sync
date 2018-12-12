package semantic

import (
	"path"

	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	"github.com/google/nomos/pkg/util/multierror"
	corev1 "k8s.io/api/core/v1"
)

// DuplicateNamespaceValidator ensures no more than one Namespace is defined in a directory.
type DuplicateNamespaceValidator struct {
	Objects []ast.FileObject
}

// Validate adds errors to the errorBuilder if there are multiple Namespaces defined in directories.
func (v DuplicateNamespaceValidator) Validate(errorBuilder *multierror.Builder) {
	namespaces := make(map[string][]ast.FileObject)

	for _, obj := range v.Objects {
		if obj.GroupVersionKind() == corev1.SchemeGroupVersion.WithKind("Namespace") {
			dir := path.Dir(obj.Source)
			namespaces[dir] = append(namespaces[dir], obj)
		}
	}

	for _, namespaces := range namespaces {
		if len(namespaces) > 1 {
			errorBuilder.Add(vet.MultipleNamespacesError{Duplicates: namespaces})
		}
	}
}
