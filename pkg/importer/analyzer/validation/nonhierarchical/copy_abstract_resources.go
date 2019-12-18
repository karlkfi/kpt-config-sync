package nonhierarchical

import (
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/kinds"
)

// CopyAbstractResources copies namespace-scoped resources without
// metadata.namespace to all declared Namespaces.
//
// Assumes that namespace-scoped resources declare EITHER metadata.namespace or
// namespace-selector. This is validated elsewhere.
func CopyAbstractResources(fileObjects []ast.FileObject) []ast.FileObject {
	// Abstract namespace-scoped resources may be selected into every Namespace.
	var namespaces []string
	for _, o := range fileObjects {
		if o.GroupVersionKind() == kinds.Namespace() {
			namespaces = append(namespaces, o.GetName())
		}
	}

	var result []ast.FileObject
	for _, o := range fileObjects {
		_, hasNamespaceSelectorAnnotation := o.GetAnnotations()[v1.NamespaceSelectorAnnotationKey]
		if !hasNamespaceSelectorAnnotation {
			// This doesn't declare a namespace-selector, so we don't need to duplicate it and can keep as-is.
			// We forbid declaring both metadata.namespace and namespace-selector, so this assumption is safe.
			result = append(result, o)
			continue
		}

		// We've already validated that namespaced objects with metadata.namespace
		// empty have the namespace-selector annotation.
		for _, namespace := range namespaces {
			nso := o.DeepCopy()
			nso.SetNamespace(namespace)
			result = append(result, nso)
		}
	}
	return result
}
