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
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
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
func (v *Copying) Error() *status.MultiError {
	return nil
}

// VisitRoot implements Visitor
func (v *Copying) VisitRoot(c *ast.Root) *ast.Root {
	nc := *c
	nc.System = c.System.Accept(v.impl)
	nc.ClusterRegistry = c.ClusterRegistry.Accept(v.impl)
	nc.Cluster = c.Cluster.Accept(v.impl)
	nc.Tree = c.Tree.Accept(v.impl)
	return &nc
}

// VisitSystem implements Visitor
func (v *Copying) VisitSystem(c *ast.System) *ast.System {
	nc := *c
	nc.Objects = nil
	for _, obj := range c.Objects {
		if newObj := obj.Accept(v.impl); newObj != nil {
			nc.Objects = append(nc.Objects, newObj)
		}
	}
	return &nc
}

// VisitSystemObject implements Visitor
func (v *Copying) VisitSystemObject(o *ast.SystemObject) *ast.SystemObject {
	return o.DeepCopy()
}

// VisitClusterRegistry implements Visitor
func (v *Copying) VisitClusterRegistry(c *ast.ClusterRegistry) *ast.ClusterRegistry {
	nc := *c
	nc.Objects = nil
	for _, obj := range c.Objects {
		if newObj := obj.Accept(v.impl); newObj != nil {
			nc.Objects = append(nc.Objects, newObj)
		}
	}
	return &nc
}

// VisitClusterRegistryObject implements Visitor
func (v *Copying) VisitClusterRegistryObject(o *ast.ClusterRegistryObject) *ast.ClusterRegistryObject {
	return o.DeepCopy()
}

// VisitCluster implements Visitor
func (v *Copying) VisitCluster(c *ast.Cluster) *ast.Cluster {
	nc := *c
	nc.Objects = nil
	for _, obj := range c.Objects {
		if newObj := obj.Accept(v.impl); newObj != nil {
			nc.Objects = append(nc.Objects, newObj)
		}
	}
	return &nc
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
