package visitor

import (
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/status"
)

// Copying is a Visitor implementation that creates and returns a copy of the
// tree.  Member functions can be overridden in order to facilitate other transforms.
//
// If you implement a Visitor based on Copying, you can use it as a base
// implementation for methods from ast.Visitor, similarly to how this was done
// in visitor.Base.  See docs for visitor.Base on how to do so.
//
// The order of traversal:
//
// 1. Root
// 2. System
// 3. SystemObjects
// 4. ClusterRegistry
// 5. ClusterRegistryObjects
// 6. Cluster
// 7. ClusterObjects
// 8. Pre-order (NLR) traversal of TreeNode(s)
//      namespaces/ for Nomos
//      hierarchy/ for Bespin
type Copying struct {
	impl ast.Visitor
}

var _ ast.Visitor = &Copying{}

// NewCopying creates a new Copying
func NewCopying() *Copying {
	cv := &Copying{}
	cv.impl = cv
	return cv
}

// SetImpl implements InheritableVisitor
func (v *Copying) SetImpl(impl ast.Visitor) {
	v.impl = impl
}

// Error implements Visitor
func (v *Copying) Error() status.MultiError {
	return nil
}

// VisitRoot implements Visitor
func (v *Copying) VisitRoot(c *ast.Root) *ast.Root {
	nc := *c
	nc.SystemObjects = nil
	nc.ClusterRegistryObjects = nil
	nc.ClusterObjects = nil
	for _, obj := range c.SystemObjects {
		if newObj := obj.Accept(v.impl); newObj != nil {
			nc.SystemObjects = append(nc.SystemObjects, newObj)
		}
	}
	for _, obj := range c.ClusterRegistryObjects {
		if newObj := obj.Accept(v.impl); newObj != nil {
			nc.ClusterRegistryObjects = append(nc.ClusterRegistryObjects, newObj)
		}
	}
	for _, obj := range c.ClusterObjects {
		if newObj := obj.Accept(v.impl); newObj != nil {
			nc.ClusterObjects = append(nc.ClusterObjects, newObj)
		}
	}
	nc.Tree = c.Tree.Accept(v.impl)
	return &nc
}

// VisitSystemObject implements Visitor
func (v *Copying) VisitSystemObject(o *ast.SystemObject) *ast.SystemObject {
	return o.DeepCopy()
}

// VisitClusterRegistryObject implements Visitor
func (v *Copying) VisitClusterRegistryObject(o *ast.ClusterRegistryObject) *ast.ClusterRegistryObject {
	return o.DeepCopy()
}

// VisitClusterObject implements Visitor
func (v *Copying) VisitClusterObject(o *ast.ClusterObject) *ast.ClusterObject {
	return o.DeepCopy()
}

// VisitTreeNode implements Visitor
func (v *Copying) VisitTreeNode(n *ast.TreeNode) *ast.TreeNode {
	// Almost-shallow copy of n, check PartialCopy to see if this is enough.
	nn := n.PartialCopy()
	nn.Objects = nil
	for _, obj := range n.Objects {
		if newObj := obj.Accept(v.impl); newObj != nil {
			nn.Objects = append(nn.Objects, newObj)
		}
	}

	nn.Children = nil
	for _, child := range n.Children {
		if ch := child.Accept(v.impl); ch != nil {
			nn.Children = append(nn.Children, ch)
		}
	}
	return nn
}

// VisitObject implements Visitor
func (v *Copying) VisitObject(o *ast.NamespaceObject) *ast.NamespaceObject {
	return o.DeepCopy()
}

// Fatal implements Visitor.
func (v *Copying) Fatal() bool {
	return v.Error() != nil
}

// RequiresValidState implements Visitor.
func (v *Copying) RequiresValidState() bool {
	return true
}
