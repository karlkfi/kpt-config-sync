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

// Node is an interface for all the nodes in the tree.  Note that this represents all types
// Visitor will visit via the Visit* methods.
type Node interface {
	// Accept is called on a visitable object to accept the visitor.  This method must call and return
	// the result of the Visit[Type] method of the visitor passed to Accept.
	Accept(v Visitor) Node
}

// Visitor allows for writing transforms on the GitContext.  The various visit methods
// will visit each type.  The return values for each Visit[Type] function are implementation dependant.
//
// When writing a new Visitor, see documentation for visitor.Base on how to
// avoid needing to implement parts of Visitor that you don't need.
//
// For visitors that are transforming the tree (based on Copying), the function should return one of
// the following:
//   Unmodified Subtree: the visitor may return the value passed to the Visit[Type] function.
//   Modified Subtree: the visitor must return a new copy of the object.
//   Deleted: the visitor should return nil to indicate deleted.
type Visitor interface {
	VisitRoot(g *Root) Node
	VisitReservedNamespaces(r *ReservedNamespaces) Node
	VisitCluster(c *Cluster) Node
	VisitClusterObjectList(o ClusterObjectList) Node
	VisitClusterObject(o *ClusterObject) Node
	VisitTreeNode(n *TreeNode) Node
	VisitObjectList(o ObjectList) Node
	VisitObject(o *NamespaceObject) Node
}

// CheckingVisitor is an interface for writing a visitor that processes the tree in some manner
// then optionally emits an error message.  This facilitates chaining visitors and stopping if one
// encounters an error.
type CheckingVisitor interface {
	Visitor

	// Error allows the visitor to emit errors that may have occurred while operating.
	Error() error
}
