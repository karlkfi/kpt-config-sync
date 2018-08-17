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

package ast

// Visitable is an interface for all the nodes in the tree.  Note that this represents all types
// Visitor will visit via the Visit* methods.
type Visitable interface {
	// Accept is called on a visitable object to accept the visitor.  The visitor in turn must then
	// call accept an child nodes to visit descendants.
	Accept(v Visitor)

	// AddChild adds a child visitable to the current visitable.  Note that some Visitables
	// may not be feasible to add to other visitables.
	AddChild(v Visitable)
}

// Visitor allows for writing transforms on the GitContext.  The various visit methods
// will visit each type.
type Visitor interface {
	VisitContext(g *Context)
	VisitReservedNamespaces(r *ReservedNamespaces)
	VisitCluster(c *Cluster)
	VisitNode(n *Node)
	VisitObject(o *Object)
}

// MutatingVisitor is an interface for writing a visitor that will modify the context
// in some manner and create a new GitContext (must copy not mutate).  MutatingVisitors
// can be chained together to compose transforms on the hierarchy.
type MutatingVisitor interface {
	Visitor

	// Result is the mutated GitContext produced by the visitor.
	Result() (*Context, error)
}
