package hydrate

import (
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/transform/selectors"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/metadata"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/validate/objects"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

// NamespaceSelectors hydrates the given Tree objects by performing namespace
// selection to filter out inactive objects that were copied into namespaces by
// the Inheritance hydrator. It also validates that objects only specify legacy
// NamespaceSelectors that are defined in an ancestor abstract namespace.
func NamespaceSelectors(objs *objects.Tree) status.MultiError {
	nsSelectors, errs := selectorMap(objs.NamespaceSelectors)
	if errs != nil {
		return errs
	}
	return visitTreeNode(objs.Tree, nsSelectors)
}

func selectorMap(objs map[string]ast.FileObject) (map[string]labels.Selector, status.MultiError) {
	var errs status.MultiError
	nsSelectors := make(map[string]labels.Selector)
	for name, obj := range objs {
		selector, err := labelSelector(obj)
		if err != nil {
			errs = status.Append(errs, err)
			continue
		}
		nsSelectors[name] = selector
	}
	return nsSelectors, errs
}

func labelSelector(obj ast.FileObject) (labels.Selector, status.Error) {
	s, sErr := obj.Structured()
	if sErr != nil {
		return nil, sErr
	}
	nss := s.(*v1.NamespaceSelector)

	selector, err := metav1.LabelSelectorAsSelector(&nss.Spec.Selector)
	if err != nil {
		return nil, selectors.InvalidSelectorError(obj, err)
	}
	if selector.Empty() {
		return nil, selectors.EmptySelectorError(obj)
	}
	return selector, nil
}

func visitTreeNode(node *ast.TreeNode, nsSelectors map[string]labels.Selector) status.MultiError {
	var errs status.MultiError
	for _, obj := range node.Objects {
		if obj.GetObjectKind().GroupVersionKind() == kinds.Namespace() {
			return applySelectors(node, obj, nsSelectors)
		}
	}

	for _, child := range node.Children {
		errs = status.Append(errs, visitTreeNode(child, nsSelectors))
	}
	return errs
}

func applySelectors(node *ast.TreeNode, namespace ast.FileObject, nsSelectors map[string]labels.Selector) status.MultiError {
	active := make(map[string]bool)
	nsLabels := labels.Set(namespace.GetLabels())
	for name, selector := range nsSelectors {
		active[name] = selector.Matches(nsLabels)
	}

	var errs status.MultiError
	var filtered []ast.FileObject
	for _, obj := range node.Objects {
		if obj.GetObjectKind().GroupVersionKind() == kinds.Namespace() {
			filtered = append(filtered, obj)
			continue
		}

		selectorName, hasAnnotation := obj.GetAnnotations()[metadata.NamespaceSelectorAnnotationKey]
		if !hasAnnotation {
			filtered = append(filtered, obj)
			continue
		}

		isActive, isKnown := active[selectorName]
		if !isKnown {
			errs = status.Append(errs, selectors.ObjectHasUnknownNamespaceSelector(obj, selectorName))
		} else if isActive {
			filtered = append(filtered, obj)
		}
	}

	if errs != nil {
		return errs
	}
	node.Objects = filtered
	return nil
}
