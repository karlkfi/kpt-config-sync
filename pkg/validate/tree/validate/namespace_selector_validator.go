package validate

import (
	"strings"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/transform/selectors"
	"github.com/google/nomos/pkg/importer/analyzer/validation/syntax"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/validate/objects"
)

// NamespaceSelector performs a second round of verification on namespace
// selector annotations. This one verifies that legacy NamespaceSelectors are
// only declared in abstract namespace and that objects only reference legacy
// NamespaceSelectors that are in an ancestor abstract namespace.
func NamespaceSelector(tree *objects.Tree) status.MultiError {
	return validateSelectorsInNode(tree.Tree, tree.NamespaceSelectors)
}

func validateSelectorsInNode(node *ast.TreeNode, nsSelectors map[string]ast.FileObject) status.MultiError {
	var objs []ast.FileObject
	for _, o := range node.Objects {
		objs = append(objs, o.FileObject)
	}
	err := validateNamespaceSelectors(objs, nsSelectors)

	for _, c := range node.Children {
		err = status.Append(err, validateSelectorsInNode(c, nsSelectors))
	}
	return err
}

// validateNamespaceSelectors returns an error if the given objects contain both
// a Namespace and one or more NamespaceSelectors.
func validateNamespaceSelectors(objs []ast.FileObject, nsSelectors map[string]ast.FileObject) status.MultiError {
	var errs status.MultiError
	var namespaceDir string
	for _, obj := range objs {
		switch obj.GroupVersionKind() {
		case kinds.Namespace():
			namespaceDir = obj.Dir().SlashPath()
		default:
			name, hasSelector := obj.GetAnnotations()[v1.NamespaceSelectorAnnotationKey]
			if hasSelector {
				selector, known := nsSelectors[name]
				if !known {
					errs = status.Append(errs, selectors.ObjectHasUnknownNamespaceSelector(obj, name))
				} else if !strings.HasPrefix(obj.SlashPath(), selector.Dir().SlashPath()) {
					// The NamespaceSelector is not in a parent directory of this object, so error.
					errs = status.Append(errs, selectors.ObjectNotInNamespaceSelectorSubdirectory(obj, selector))
				}
			}
		}
	}
	if namespaceDir != "" {
		selectorsInNode := selectorsInNamespaceDir(namespaceDir, nsSelectors)
		if len(selectorsInNode) > 0 {
			errs = status.Append(errs, syntax.IllegalKindInNamespacesError(selectorsInNode...))
		}
	}
	return errs
}

func selectorsInNamespaceDir(dir string, nsSelectors map[string]ast.FileObject) []id.Resource {
	var result []id.Resource
	for _, obj := range nsSelectors {
		if obj.Dir().SlashPath() == dir {
			result = append(result, obj)
		}
	}
	return result
}
