package validate

import (
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/transform"
	"github.com/google/nomos/pkg/importer/analyzer/validation/semantic"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/validate/objects"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Inheritance verifies that all syncable resources in an abstract namespace
// have a concrete Namespace as a descendant.
func Inheritance(tree *objects.Tree) status.MultiError {
	_, err := validateTreeNode(tree.Tree)
	return err
}

// validateTreeNode returns True if the give node is a Namespace or if any of
// its descendants are.
func validateTreeNode(node *ast.TreeNode) (bool, status.MultiError) {
	hasSyncableObjects := false
	var namespaces []client.Object
	for _, obj := range node.Objects {
		if obj.GetObjectKind().GroupVersionKind() == kinds.Namespace() {
			namespaces = append(namespaces, obj)
		} else if !transform.IsEphemeral(obj.GetObjectKind().GroupVersionKind()) {
			hasSyncableObjects = true
		}
	}

	if len(namespaces) > 1 {
		return true, status.MultipleSingletonsError(namespaces...)
	}
	if len(namespaces) == 1 {
		return true, nil
	}

	var errs status.MultiError
	foundDescendant := false
	for _, child := range node.Children {
		hasNamespaceDescendant, err := validateTreeNode(child)
		foundDescendant = foundDescendant || hasNamespaceDescendant
		errs = status.Append(errs, err)
	}
	if hasSyncableObjects {
		if len(node.Children) == 0 {
			errs = status.Append(errs, semantic.UnsyncableResourcesInLeaf(node))
		} else if !foundDescendant {
			errs = status.Append(errs, semantic.UnsyncableResourcesInNonLeaf(node))
		}
	}
	return foundDescendant, errs
}
