package selectors

import (
	"strings"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	"k8s.io/apimachinery/pkg/labels"
)

// ResolveHierarchicalNamespaceSelectors returns the list of objects that should by synced given
// the set of declared NamespaceSelectors for hierarchical repos.
//
// This function assumes namespace-scoped objects have already been copied down from their
// AbstractNamespaces into the Namespaces in the child subdirectories.
//
// Rules:
// 1. A NamespaceSelector may select any Namespace in its child directories.
// 2. A NamespaceSelector selects all Namespaces matching its LabelSelector.
// 3. If an object declares the namespace-selector annotation and is not in a selected Namespace, it is inactive.
// 4. Otherwise, the object is active.
//
// Returns error if
// - A NamespaceSelector is invalid
// - An object references a NamespaceSelector that does not exist.
// - An object references a NamespaceSelector in a non-parent directory.
func ResolveHierarchicalNamespaceSelectors(objects []ast.FileObject) ([]ast.FileObject, status.MultiError) {
	nssStates, errs := getHierarchicalNamespaceSelectorStates(objects)
	errs = status.Append(errs, validateHierarchicalSelectorReferences(objects))
	if errs != nil {
		// Either a NamespaceSelector was invalid or the Selector references were wrong,
		// so exit early.
		return nil, errs
	}

	return resolveNamespaceSelectors(nssStates, objects)
}

// getHierarchicalNamespaceSelectorStates returns a map from each defined NamespaceSelector name
// to a map from each Namespace in a child directory of the NamespaceSelector's directory.
//
// This cache means we don't have to resolve Namespace-selection for every object, just once per
// Namespace.
func getHierarchicalNamespaceSelectorStates(objects []ast.FileObject) (map[selectorName]map[namespaceName]state, status.MultiError) {
	selectors, errs := getNamespaceSelectors(objects)
	if errs != nil {
		// Problem parsing NamespaceSelectors, son don't try to continue.
		return nil, errs
	}
	namespaces := filterNamespaces(objects)

	nssStates := make(map[selectorName]map[namespaceName]state)
	for _, selector := range selectors {
		// Get the directory of the NamespaceSelector. We need a record of all Namespaces
		// in child directories of this directory.
		selectorDir := selector.Path.Dir().SlashPath()

		// Make a map to record whether this NamespaceSelector selects each Namespace.
		namespaceStates := make(map[namespaceName]state)
		for _, namespace := range namespaces {
			namespacePath := namespace.SlashPath()
			if !strings.HasPrefix(namespacePath, selectorDir) {
				// This NamespaceSelector does not apply to this Namespace as it does not
				// appear in any of its AbstractNamespace parents.
				continue
			}

			if selector.Matches(labels.Set(namespace.GetLabels())) {
				namespaceStates[namespaceName(namespace.GetName())] = active
			} else {
				namespaceStates[namespaceName(namespace.GetName())] = inactive
			}
		}

		nssStates[selectorName(selector.GetName())] = namespaceStates
	}

	return nssStates, errs
}

// getNamespaceSelectors returns the list of NamespaceSelectors in the passed array of FileObjects.
//
// Returns error if objects contains an invalid NamespaceSelector.
func getNamespaceSelectors(objects []ast.FileObject) ([]selectorFileObject, status.MultiError) {
	var namespaceSelectors []selectorFileObject
	var errs status.MultiError
	for _, object := range objects {
		if o, ok := object.Object.(*v1.NamespaceSelector); ok {
			selector, err := asSelectorFileObject(object, o.Spec.Selector)
			if err != nil {
				errs = status.Append(errs, err)
				continue
			}
			namespaceSelectors = append(namespaceSelectors, selector)
		}
	}
	return namespaceSelectors, errs
}

// filterNamespaces returns the list of FileObjects which are Namespaces.
func filterNamespaces(objects []ast.FileObject) []ast.FileObject {
	var result []ast.FileObject
	for _, object := range objects {
		if object.GroupVersionKind() == kinds.Namespace() {
			result = append(result, object)
		}
	}
	return result
}

// resolveNamespaceSelectors implements the core flat Namespace-selection logic.
//
// nssStates is a map from each NamespaceSelector's name to a map of whether is selects each Namespace.
// objects is the list of objects to resolve Namespace-selection on.
//
// Returns an error if an object declares an unknown NamespaceSelector.
// Assumes value in nssStates contains an entry for every Namespace that may declare that NamespaceSelector.
//   Otherwise, returns an InternalError.
func resolveNamespaceSelectors(nssStates map[selectorName]map[namespaceName]state, objects []ast.FileObject) ([]ast.FileObject, status.MultiError) {
	var result []ast.FileObject
	var errs status.MultiError
	for _, object := range objects {
		if object.GroupVersionKind() == kinds.NamespaceSelector() {
			// Discard NamespaceSelectors as we don't need them anymore.
			continue
		}

		objState, err := objectNamespaceSelectorState(nssStates, object)
		if err != nil {
			errs = status.Append(errs, err)
		}
		if objState == active {
			// The object is active in its Namespace, so keep it.
			result = append(result, object)
		}
	}

	return result, errs
}

// objectNamespaceSelectorState returns whether the object is active in the Namespace. This is determined by
//
// 1. If the object declares a namespace-selector annotation which is inactive in its Namespace, it is inactive.
// 2. Otherwise, the object is active.
//
// Returns an error if the object references an undeclared NamespaceSelector.
//
// Assumes each entry of nssStates has a state for every Namespace for which that NamespaceSelector
// might apply. Otherwise, returns an InternalError.
func objectNamespaceSelectorState(nssStates map[selectorName]map[namespaceName]state, object ast.FileObject) (state, status.Error) {
	// NamespaceSelectors only work on Namespace-scoped objects. Cluster-scoped objects return empty
	// string for GetNamespace().
	if object.GetNamespace() == "" {
		return active, nil
	}

	// The namespace-selector annotation, if defined, is the name of the NamespaceSelector it references.
	nsSelectorName, hasAnnotation := object.GetAnnotations()[v1.NamespaceSelectorAnnotationKey]
	// Objects with no namespace-selector nssName are implicitly active.
	if !hasAnnotation {
		return active, nil
	}

	selectorState, selectorDefined := nssStates[selectorName(nsSelectorName)]
	if !selectorDefined {
		// We require that all objects which declare the namespace-selector annotation reference
		// a NamespaceSelector that exists.
		return unknown, ObjectHasUnknownNamespaceSelector(object, nsSelectorName)
	}

	nsState, hasDefinedState := selectorState[namespaceName(object.GetNamespace())]
	if !hasDefinedState {
		// This error should never happen.
		return unknown, status.InternalErrorf("NamespaceSelector %q has no defined state for Namespace %q",
			nsSelectorName, object.GetNamespace())
	}
	return nsState, nil
}

// validateHierarchicalSelectorReferences returns errors if objects declare non-existent
// NamespaceSelectors, or if they declare NamespaceSelectors in non-parent directories.
func validateHierarchicalSelectorReferences(objects []ast.FileObject) status.MultiError {
	selectors := make(map[selectorName]ast.FileObject)
	for _, o := range objects {
		if o.GroupVersionKind() == kinds.NamespaceSelector() {
			selectors[selectorName(o.GetName())] = o
		}
	}

	var errs status.MultiError
	for _, o := range objects {
		annotation, hasAnnotation := o.GetAnnotations()[v1.NamespaceSelectorAnnotationKey]
		if !hasAnnotation {
			// No namespace-selector annotation, so nothing to validate.
			continue
		}

		selector, selectorExists := selectors[selectorName(annotation)]
		if !selectorExists {
			// The NamespaceSelector does not exist, so error.
			errs = status.Append(errs, ObjectHasUnknownNamespaceSelector(o, annotation))
			continue
		}

		selectorDir := selector.Dir().SlashPath()
		if !strings.HasPrefix(o.SlashPath(), selectorDir) {
			// The NamespaceSelector is not in a parent directory of this object, so error.
			errs = status.Append(errs, ObjectNotInNamespaceSelectorSubdirectory(o, selector))
		}
	}
	return errs
}

// ObjectHasUnknownNamespaceSelector reports that `resource`'s namespace-selector annotation
// references a NamespaceSelector that does not exist.
func ObjectHasUnknownNamespaceSelector(resource id.Resource, selector string) status.Error {
	return objectHasUnknownSelector.
		Sprintf("Resource %q MUST refer to an existing NamespaceSelector, but has annotation %s=%q which maps to no declared NamespaceSelector",
			resource.GetName(), v1.NamespaceSelectorAnnotationKey, selector).
		BuildWithResources(resource)
}

// ObjectNotInNamespaceSelectorSubdirectory reports that `resource` is not in a subdirectory of the directory
// declaring `selector`.
func ObjectNotInNamespaceSelectorSubdirectory(resource id.Resource, selector id.Resource) status.Error {
	return objectHasUnknownSelector.
		Sprintf("Resource %q MUST refer to a NamespaceSelector in its directory or a parent directory. "+
			"Either remove the annotation %s=%q from %q or move NamespaceSelector %q to a parent directory of %q.",
			resource.GetName(), v1.NamespaceSelectorAnnotationKey, selector.GetName(), resource.GetName(), selector.GetName(), resource.GetName()).
		BuildWithResources(selector, resource)
}
