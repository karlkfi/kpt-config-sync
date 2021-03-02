package validate

import (
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/validation/syntax"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/validate/objects"
)

// NamespaceSelector performs a second round of verification on namespace
// selector annotations. This one verifies that legacy NamespaceSelectors are
// only declared in abstract namespaces.
func NamespaceSelector(tree *objects.Tree) status.MultiError {
	return validateSelectorsInNode(tree.Tree)
}

func validateSelectorsInNode(node *ast.TreeNode) status.MultiError {
	var objs []ast.FileObject
	for _, o := range node.Objects {
		objs = append(objs, o.FileObject)
	}
	err := validateNamespaceSelectors(objs)

	for _, c := range node.Children {
		err = status.Append(err, validateSelectorsInNode(c))
	}
	return err
}

// validateNamespaceSelectors returns an error if the given objects contain both
// a Namespace and one or more NamespaceSelectors.
func validateNamespaceSelectors(objs []ast.FileObject) status.MultiError {
	var selectors []id.Resource
	var isNamespace bool
	for _, obj := range objs {
		switch obj.GroupVersionKind() {
		case kinds.Namespace():
			isNamespace = true
		case kinds.NamespaceSelector():
			selectors = append(selectors, obj)
		}
	}
	if isNamespace && len(selectors) > 0 {
		return syntax.IllegalKindInNamespacesError(selectors...)
	}
	return nil
}
