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

import "github.com/google/nomos/pkg/policyimporter/analyzer/ast"

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
// 2. Cluster
// 3. ReservedNamespaces
// 4. Pre-order traversal of TreeNode(s)
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
func (v *Copying) Error() error {
	return nil
}

// VisitRoot implements Visitor
func (v *Copying) VisitRoot(c *ast.Root) *ast.Root {
	nc := *c
	nc.Cluster = c.Cluster.Accept(v.impl)
	nc.Tree = c.Tree.Accept(v.impl)
	return &nc
}

// VisitCluster implements Visitor
func (v *Copying) VisitCluster(c *ast.Cluster) *ast.Cluster {
	nc := *c
	nc.Objects = c.Objects.Accept(v.impl)
	return &nc
}

// VisitClusterObjectList implements Visitor
func (v *Copying) VisitClusterObjectList(objectList ast.ClusterObjectList) ast.ClusterObjectList {
	var olc ast.ClusterObjectList
	for _, object := range objectList {
		if obj := object.Accept(v.impl); obj != nil {
			olc = append(olc, obj)
		}
	}
	return olc
}

// VisitClusterObject implements Visitor
func (v *Copying) VisitClusterObject(o *ast.ClusterObject) *ast.ClusterObject {
	return o.DeepCopy()
}

// VisitObjectList implements Visitor
func (v *Copying) VisitObjectList(objectList ast.ObjectList) ast.ObjectList {
	var olc ast.ObjectList
	for _, object := range objectList {
		if obj := object.Accept(v.impl); obj != nil {
			olc = append(olc, obj)
		}
	}
	return olc
}

// VisitTreeNode implements Visitor
func (v *Copying) VisitTreeNode(n *ast.TreeNode) *ast.TreeNode {
	// Almost-shallow copy of n, check PartialCopy to see if this is enough.
	nn := n.PartialCopy()
	nn.Objects = n.Objects.Accept(v.impl)
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
