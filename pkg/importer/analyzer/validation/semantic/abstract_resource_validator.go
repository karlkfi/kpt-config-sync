package semantic

import (
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/ast/node"
	"github.com/google/nomos/pkg/importer/analyzer/transform"
	"github.com/google/nomos/pkg/importer/analyzer/vet"
	"github.com/google/nomos/pkg/importer/analyzer/visitor"
	"github.com/google/nomos/pkg/status"
)

// NewAbstractResourceValidator returns a Visitor that validates there are no resources in an Abstract
// Namespace directory that doesn't have any Namespace directory children.
func NewAbstractResourceValidator() ast.Visitor {
	return visitor.NewRootValidator(func(g *ast.Root) *status.MultiError {
		n := g.Tree
		if n == nil {
			return nil
		}

		var errs status.ErrorBuilder
		validate(n, hasSyncableObjects(n) && n.Type == node.AbstractNamespace, &errs)
		return errs.Build()
	})
}

func validate(n *ast.TreeNode, requiresNamespaceDescendant bool, errs *status.ErrorBuilder) bool {
	if requiresNamespaceDescendant {
		// We require that the node or one of its descendants node contains a Namespace; so look for one.
		if n.Type == node.Namespace {
			// We found a namespace: back out.
			return true
		}

		if len(n.Children) == 0 {
			// Handle validating a leaf node with a problematic ancestor.
			if hasSyncableObjects(n) {
				// We've found another problematic abstract namespace leaf node below the one requiring a namespace descendant.
				// That descendant node supersedes the ancestor. So, return an error for the descendant and omit any ancestor
				// error.
				errs.Add(vet.UnsyncableResourcesError{Dir: n})
				return true
			}
			// No valid descendant. Propagate this up to the problematic ancestor.
			return false
		}

		// Handle validating an intermediate node with a problematic ancestor.
		oneValidChild := false
		for _, c := range n.Children {
			if validate(c, requiresNamespaceDescendant, errs) {
				oneValidChild = true
			}
		}
		if oneValidChild {
			// We found a valid descendant. Ensure the problematic ancestor doesn't generate an error.
			return true
		}

		// We didn't find any descendants with a Namespace. Generate an error and ensure any problematic ancestors don't
		// also generate an error.
		errs.Add(vet.UnsyncableResourcesError{Dir: n, Ancestor: true})
		return true
	}

	for _, c := range n.Children {
		// The node is not problematic, but we still need to check its descendants for issues.
		validate(c, c.Type == node.AbstractNamespace && hasSyncableObjects(c), errs)
	}
	return true
}

func hasSyncableObjects(n *ast.TreeNode) bool {
	for _, o := range n.Objects {
		if !transform.IsEphemeral(o.GroupVersionKind()) {
			return true
		}
	}
	return false
}
