package semantic

import (
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/ast/node"
	"github.com/google/nomos/pkg/importer/analyzer/transform"
	"github.com/google/nomos/pkg/importer/analyzer/visitor"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/status"
)

// NewAbstractResourceValidator returns a Visitor that validates there are no resources in an Abstract
// Namespace directory that doesn't have any Namespace directory children.
func NewAbstractResourceValidator() ast.Visitor {
	return visitor.NewRootValidator(func(g *ast.Root) status.MultiError {
		n := g.Tree
		if n == nil {
			return nil
		}

		_, errs := validate(n, hasSyncableObjects(n) && n.Type == node.AbstractNamespace)
		return errs
	})
}

func validate(n *ast.TreeNode, requiresNamespaceDescendant bool) (bool, status.MultiError) {
	var errs status.MultiError
	if requiresNamespaceDescendant {
		// We require that the node or one of its descendants node contains a Namespace; so look for one.
		if n.Type == node.Namespace {
			// We found a namespace: back out.
			return true, errs
		}

		if len(n.Children) == 0 {
			// Handle validating a leaf node with a problematic ancestor.
			if hasSyncableObjects(n) {
				// We've found another problematic abstract namespace leaf node below the one requiring a namespace descendant.
				// That descendant node supersedes the ancestor. So, return an error for the descendant and omit any ancestor
				// error.
				errs = status.Append(errs, UnsyncableResourcesInLeaf(n))
				return true, errs
			}
			// No valid descendant. Propagate this up to the problematic ancestor.
			return false, errs
		}

		// Handle validating an intermediate node with a problematic ancestor.
		oneValidChild := false
		for _, c := range n.Children {
			isValidChild, newErrs := validate(c, requiresNamespaceDescendant)
			oneValidChild = oneValidChild || isValidChild
			errs = status.Append(errs, newErrs)
		}
		if oneValidChild {
			// We found a valid descendant. Ensure the problematic ancestor doesn't generate an error.
			return true, errs
		}

		// We didn't find any descendants with a Namespace. Generate an error and ensure any problematic ancestors don't
		// also generate an error.
		errs = status.Append(errs, UnsyncableResourcesInNonLeaf(n))
		return true, errs
	}

	for _, c := range n.Children {
		// The node is not problematic, but we still need to check its descendants for issues.
		_, newErrs := validate(c, c.Type == node.AbstractNamespace && hasSyncableObjects(c))
		errs = status.Append(errs, newErrs)
	}
	return true, errs
}

func hasSyncableObjects(n *ast.TreeNode) bool {
	for _, o := range n.Objects {
		if !transform.IsEphemeral(o.GroupVersionKind()) {
			return true
		}
	}
	return false
}

// UnsyncableResourcesErrorCode is the error code for UnsyncableResourcesError
const UnsyncableResourcesErrorCode = "1044"

var unsyncableResourcesError = status.NewErrorBuilder(UnsyncableResourcesErrorCode)

// UnsyncableResourcesInLeaf reports that a leaf node has resources but is not a Namespace.
func UnsyncableResourcesInLeaf(dir id.TreeNode) status.Error {
	return unsyncableResourcesError.WithPaths(dir).Errorf(
		"An %[1]s directory with configs MUST have at least one %[2]s subdirectory. "+
			"To fix, do one of the following: add a %[2]s directory below %[3]q, "+
			"add a Namespace config to %[3]q, "+
			"or remove the configs in %[3]q:", node.AbstractNamespace, node.Namespace, dir.Name())
}

// UnsyncableResourcesInNonLeaf reports that a node has resources and descendants, but none of its
// descendants are Namespaces.
func UnsyncableResourcesInNonLeaf(dir id.TreeNode) status.Error {
	return unsyncableResourcesError.WithPaths(dir).Errorf(
		"An %[1]s directory with configs MUST have at least one %[2]s subdirectory. "+
			"To fix, do one of the following: add a %[2]s directory below %[3]q, "+
			"convert a directory below to a %[2]s directory, "+
			"or remove the configs in %[3]q:", node.AbstractNamespace, node.Namespace, dir.Name())
}
