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

// Base implements visiting all children for a visitor (like a base class).
// Derived children need to have a Base and invoke base.VisitX(x) to continue
// visiting children (like calling a base class method).
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
	vb.impl = impl
}

// VisitContext implements Visitor
func (vb *Base) VisitContext(g *ast.Context) {
	if g.Cluster != nil {
		g.Cluster.Accept(vb.impl)
	}
	if g.ReservedNamespaces != nil {
		g.ReservedNamespaces.Accept(vb.impl)
	}
	if g.Tree != nil {
		g.Tree.Accept(vb.impl)
	}
}

// VisitReservedNamespaces implements Visitor
func (vb *Base) VisitReservedNamespaces(r *ast.ReservedNamespaces) {
	// leaf - noop
}

// VisitCluster implements Visitor
func (vb *Base) VisitCluster(c *ast.Cluster) {
	for _, object := range c.Objects {
		object.Accept(vb.impl)
	}
}

// VisitNode implements Visitor
func (vb *Base) VisitNode(n *ast.Node) {
	for _, obj := range n.Objects {
		obj.Accept(vb.impl)
	}
	for _, child := range n.Children {
		child.Accept(vb.impl)
	}
}

// VisitObject implements Visitor
func (vb *Base) VisitObject(o *ast.Object) {
	// leaf - noop
}
