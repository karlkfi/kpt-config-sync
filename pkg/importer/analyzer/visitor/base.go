package visitor

import (
	"github.com/davecgh/go-spew/spew"
	"github.com/golang/glog"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/status"
)

// Base implements visiting all children for a visitor (like a base class).
// Derived children need to have a Base and invoke base.VisitX(x) to continue
// visiting children (like calling a base class method).  This removes the need
// for every new visitor to implement all methods in ast.Visitor.
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
//
// Pre-order traversal guarantees that when a child node is visited that all of its ancestors have
// been visited.
//
// Example:
//      type myVisitor {
//        // Base supplies default implementations of all Visitor methods.
//        // No need to implement methods that you don't need a custom implementation for.
//        *visitor.Base
//
//        // See VisitTreeNode in this example.
//        ancestors []*ast.TreeNode
//      }
//
//      func NewMyVisitor() myVisitor {
//        v := &myVisitor{Base: visitor.NewBase()}
//        v.SetImpl(v)
//        return v
//      }
//
//      func (v *myVisitor) VisitRoot(ctx *ast.Root) *ast.Root {
//        // Do whatever you need to do in this Visitor that needs to happen before
//        // traversing children.
//        // Then call the matching continuation method from Base. In this case it is
//        // VisitRoot. If the corresponding Base method is _not_ called, Visit*
//        // functions on remaining elements will not be automatically called.
//        return v.Base.VisitRoot(ctx)
//      }
//
//      func (v *myVisitor) VisitTreeNode(n *ast.TreeNode) *ast.TreeNode {
//        // You can create a context stack by adding logic after visiting children.
//        ancestors = append(ancestors, n)
//        result := v.Base.VisitTreeNode(n)
//        // Then do whatever cleanup you need to do after traversing children.
//        ancestors = ancestors[:len(ancestors)-1]
//        return result
//      }
type Base struct {
	// impl handles the upper most implementation for the visitor.  This allows VisitorBase
	// to return control to the top object in the visitor chain.
	impl ast.Visitor
}

var _ ast.Visitor = &Base{}

// NewBase creates a new VisitorBase.
func NewBase() *Base {
	return &Base{}
}

// SetImpl sets the impl for VisitorBase.  This would be part of the constructor except that
// it would lead to a circular dependency and it makes the most sense for the upper most
// object to set the impl value.
func (vb *Base) SetImpl(impl ast.Visitor) {
	if vb == nil {
		panic(status.InternalError("Base is nil. Allocate Base before using it."))
	}
	vb.impl = impl
}

// VisitRoot implements Visitor
func (vb *Base) VisitRoot(g *ast.Root) *ast.Root {
	if glog.V(5) {
		glog.Infof("VisitRoot(): ENTER: %v", spew.Sdump(g))
	}
	defer glog.V(6).Infof("VisitRoot(): EXIT")
	for _, o := range g.SystemObjects {
		o.Accept(vb.impl)
	}
	for _, o := range g.ClusterRegistryObjects {
		o.Accept(vb.impl)
	}
	for _, o := range g.ClusterObjects {
		o.Accept(vb.impl)
	}
	g.Tree.Accept(vb.impl)
	return g
}

// VisitSystemObject implements Visitor
func (vb *Base) VisitSystemObject(o *ast.SystemObject) *ast.SystemObject {
	// leaf - noop
	return o
}

// VisitClusterRegistryObject implements Visitor
func (vb *Base) VisitClusterRegistryObject(o *ast.ClusterRegistryObject) *ast.ClusterRegistryObject {
	// leaf - noop
	return o
}

// VisitClusterObject implements Visitor
func (vb *Base) VisitClusterObject(o *ast.ClusterObject) *ast.ClusterObject {
	// leaf - noop
	return o
}

// VisitTreeNode implements Visitor
func (vb *Base) VisitTreeNode(n *ast.TreeNode) *ast.TreeNode {
	if glog.V(5) {
		glog.Infof("VisitTreeNode(): ENTER: %v", spew.Sdump(n))
	}
	defer glog.V(6).Infof("VisitTreeNode(): EXIT")
	for _, o := range n.Objects {
		o.Accept(vb.impl)
	}
	for _, child := range n.Children {
		child.Accept(vb.impl)
	}
	return n
}

// VisitObject implements Visitor
func (vb *Base) VisitObject(o *ast.NamespaceObject) *ast.NamespaceObject {
	if glog.V(5) {
		glog.Infof("VisitObject(): ENTER: %+v", spew.Sdump(o))
	}
	// leaf - noop
	return o
}

// Error implements Visitor.
func (vb *Base) Error() status.MultiError {
	return nil
}

// Fatal implements Visitor.
func (vb *Base) Fatal() bool {
	return false
}

// RequiresValidState implements Visitor.
func (vb *Base) RequiresValidState() bool {
	return false
}
