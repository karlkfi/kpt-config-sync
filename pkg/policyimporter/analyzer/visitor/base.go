/*
Copyright 2018 The Nomos Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package visitor

import (
	"github.com/davecgh/go-spew/spew"
	"github.com/golang/glog"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
)

// Base implements visiting all children for a visitor (like a base class).
// Derived children need to have a Base and invoke base.VisitX(x) to continue
// visiting children (like calling a base class method).  This removes the need
// for every new visitor to implement all methods in ast.Visitor.
//
// The order of traversal:
//
// 1. Root
// 2. Cluster
// 3. ReservedNamespaces
// 4. Pre-order traversal of TreeNode(s)
//
// Example:
//      type myVisitor {
//        // Base supplies default implementations of all Visitor methods.
//        // No need to implement methods that you don't need a custom implementation for.
//        *visitor.Base
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
//        // Then call the matching continuation method from Base, in this case it is
//        // VisitRoot.
//        return v.Base.VisitRoot(ctx)
//      }
type Base struct {
	// impl handles the upper most implementation for the visitor.  This allows VisitorBase
	// to return control to the top object in the visitor chain.
	impl ast.Visitor
}

// NewBase creates a new VisitorBase.
func NewBase() *Base {
	return &Base{}
}

// SetImpl sets the impl for VisitorBase.  This would be part of the constructor except that
// it would lead to a circular dependency and it makes the most sense for the upper most
// object to set the impl value.
func (vb *Base) SetImpl(impl ast.Visitor) {
	vb.impl = impl
}

// VisitRoot implements Visitor
func (vb *Base) VisitRoot(g *ast.Root) *ast.Root {
	if glog.V(5) {
		glog.Infof("VisitRoot(): ENTER: %v", spew.Sdump(g))
	}
	defer glog.V(6).Infof("VisitRoot(): EXIT")
	g.System.Accept(vb.impl)
	g.ClusterRegistry.Accept(vb.impl)
	g.Cluster.Accept(vb.impl)
	g.Tree.Accept(vb.impl)
	return g
}

// VisitSystem implements Visitor
func (vb *Base) VisitSystem(c *ast.System) *ast.System {
	for _, o := range c.Objects {
		o.Accept(vb.impl)
	}
	return c
}

// VisitSystemObject implements Visitor
func (vb *Base) VisitSystemObject(o *ast.SystemObject) *ast.SystemObject {
	// leaf - noop
	return o
}

// VisitClusterRegistry implements Visitor
func (vb *Base) VisitClusterRegistry(c *ast.ClusterRegistry) *ast.ClusterRegistry {
	for _, o := range c.Objects {
		o.Accept(vb.impl)
	}
	return c
}

// VisitClusterRegistryObject implements Visitor
func (vb *Base) VisitClusterRegistryObject(o *ast.ClusterRegistryObject) *ast.ClusterRegistryObject {
	// leaf - noop
	return o
}

// VisitCluster implements Visitor
func (vb *Base) VisitCluster(c *ast.Cluster) *ast.Cluster {
	for _, o := range c.Objects {
		o.Accept(vb.impl)
	}
	return c
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

func (vb *Base) Error() error {
	return nil
}
