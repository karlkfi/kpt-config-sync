package transform

import (
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/ast/node"
	"github.com/google/nomos/pkg/importer/analyzer/visitor"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
)

// nodeContext keeps track of objects during the tree traversal for purposes of inheriting values.
type nodeContext struct {
	nodeType  node.Type              // the type of node being processed
	nodePath  cmpath.Path            // the node's path, used for annotating inherited objects
	inherited []*ast.NamespaceObject // the objects that are inherited from the node.
}

// InheritanceSpec defines the spec for inherited resources.
type InheritanceSpec struct {
	Mode v1.HierarchyModeType
}

// InheritanceVisitor aggregates hierarchical objects.
type inheritanceVisitor struct {
	// Copying is used for copying parts of the ast.Root tree and continuing underlying visitor iteration.
	*visitor.Copying
	// treeContext is a stack that tracks ancestry and inherited objects during the tree traversal.
	treeContext []nodeContext
}

var _ ast.Visitor = &inheritanceVisitor{}

// NewInheritanceVisitor returns a new InheritanceVisitor for the given GroupKind
func NewInheritanceVisitor() ast.Visitor {
	iv := &inheritanceVisitor{
		Copying: visitor.NewCopying(),
	}
	iv.SetImpl(iv)
	return iv
}

// Error implements Visitor
func (v *inheritanceVisitor) Error() status.MultiError {
	return nil
}

// VisitTreeNode implements Visitor
//
// Copies inherited objects into their Namespaces. Otherwise mutating the object later in any
// individual object modifies all copies in other Namespaces.
func (v *inheritanceVisitor) VisitTreeNode(n *ast.TreeNode) *ast.TreeNode {
	v.treeContext = append(v.treeContext, nodeContext{
		nodeType: n.Type,
		nodePath: n.Path,
	})
	newNode := v.Copying.VisitTreeNode(n)
	v.treeContext = v.treeContext[:len(v.treeContext)-1]
	if n.Type == node.Namespace {
		for _, ctx := range v.treeContext {
			for _, inherited := range ctx.inherited {
				newNode.Objects = append(newNode.Objects, &ast.NamespaceObject{FileObject: inherited.DeepCopy()})
			}
		}
	}
	return newNode
}

// VisitObject implements Visitor
func (v *inheritanceVisitor) VisitObject(o *ast.NamespaceObject) *ast.NamespaceObject {
	context := &v.treeContext[len(v.treeContext)-1]
	if context.nodeType == node.AbstractNamespace {
		if o.GroupVersionKind() != kinds.NamespaceSelector() {
			// Don't copy down NamespaceSelectors.
			context.inherited = append(context.inherited, o)
		}
	}
	return o
}
