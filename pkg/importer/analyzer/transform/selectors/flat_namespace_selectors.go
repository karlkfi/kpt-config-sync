package selectors

import (
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
)

// ResolveFlatNamespaceSelectors returns the list of objects returned after
// resolving NamespaceSelectors in a non-hierarchical repository.
//
// This only removes objects declaring a namespace-selector that is inactive in
// the declared metadata.namespace.
func ResolveFlatNamespaceSelectors(objects []ast.FileObject) ([]ast.FileObject, status.MultiError) {
	nssStates, errs := getNamespaceSelectorStates(objects)
	errs = status.Append(errs, validateFlatSelectorReferences(objects))
	if errs != nil {
		return nil, errs
	}

	return resolveNamespaceSelectors(nssStates, objects)
}

// validateFlatSelectorReferences returns errors if objects reference non-existent
// NamespaceSelectors.
func validateFlatSelectorReferences(objects []ast.FileObject) status.MultiError {
	selectorExists := make(map[selectorName]bool)
	selectors, errs := getNamespaceSelectors(objects)
	if errs != nil {
		return errs
	}

	for _, selector := range selectors {
		selectorExists[selectorName(selector.GetName())] = true
	}

	for _, o := range objects {
		if o.GroupVersionKind() == kinds.NamespaceSelector() {
			continue
		}

		annotation, hasAnnotation := o.GetAnnotations()[v1.NamespaceSelectorAnnotationKey]
		if !hasAnnotation {
			continue
		}
		if !selectorExists[selectorName(annotation)] {
			errs = status.Append(errs, ObjectHasUnknownNamespaceSelector(o, annotation))
		}
	}
	return errs
}
