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
// The order of traversal:
//
// 1. Context
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

// Result implements Visitor
func (v *Copying) Result() error {
	return nil
}

// VisitContext implements Visitor
func (v *Copying) VisitContext(c *ast.Context) ast.Node {
	nc := *c
	nc.Cluster, _ = c.Cluster.Accept(v.impl).(*ast.Cluster)
	nc.ReservedNamespaces, _ = c.ReservedNamespaces.Accept(v.impl).(*ast.ReservedNamespaces)
	nc.Tree, _ = c.Tree.Accept(v.impl).(*ast.TreeNode)
	return &nc
}

// VisitReservedNamespaces implements Visitor
func (v *Copying) VisitReservedNamespaces(r *ast.ReservedNamespaces) ast.Node {
	return r.DeepCopy()
}

// VisitCluster implements Visitor
func (v *Copying) VisitCluster(c *ast.Cluster) ast.Node {
	nc := *c
	nc.Objects, _ = c.Objects.Accept(v.impl).(ast.ObjectList)
	return &nc
}

// VisitObjectList implements Visitor
func (v *Copying) VisitObjectList(objectList ast.ObjectList) ast.Node {
	var olc ast.ObjectList
	for _, object := range objectList {
		if obj, ok := object.Accept(v.impl).(*ast.Object); ok {
			olc = append(olc, obj)
		}
	}
	return olc
}

// VisitTreeNode implements Visitor
func (v *Copying) VisitTreeNode(n *ast.TreeNode) ast.Node {
	nn := *n
	nn.Objects, _ = n.Objects.Accept(v.impl).(ast.ObjectList)
	nn.Children = nil
	for _, child := range n.Children {
		if ch, ok := child.Accept(v.impl).(*ast.TreeNode); ok {
			nn.Children = append(nn.Children, ch)
		}
	}
	return &nn
}

// VisitObject implements Visitor
func (v *Copying) VisitObject(o *ast.Object) ast.Node {
	return o.DeepCopy()
}
