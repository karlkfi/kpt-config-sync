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

/*
Package visitor defines two useful visitors that can be leveraged by other visitors.

Base is a simple visitor that visits all nodes.  It approximates a base class for other
visitors to use for visiting the tree.  Example usage of the Base visitor
can be found in the Copying visitor.

The Copying visitor makes a copy of the AST while visiting.  Visitors that mutate the tree
can re-use the copying logic in order to facilitate a copy-on-update strategy for the tree.
Copying can be omitted by returning the node in the visit method instead of calling Copying.Visit[Type].
Removing the node from the tree can performed by returning nil from the Visit[Type] method.
Any Visitors mutating the tree MUST copy any node that has updates in its child nodes.

Validating Visitors
Writing a validating visitor entails using Base to visit the tree and returning the original ast.Context
from the VisitContext method (effectively an identity operation).  Pass / fail should be returned
through the Result() method of the visitor to fulfill the Visitor contract.

Mutating Visitors
For visitors that want to mutate the tree they will need to treat the tree as immutable and update
via copy on write.  This means that for any object that needs to be updated, a shallow copy must be
made then the field should be updated on the shallow copy.  This means changing a value in a subtree
will require updating all ancestors up to the root node.  The rationale is to preserve a view of the
tree before and after each visitor pass to facilitate debugging for visitor writers as well as
anyone who is authoring policies.

Example usage:
	// RemoveNamespaceTesting will remove any namespace scoped resource with a "testing=true" annotation.
	type RemoveNamespaceTesting struct {
		// base is the base copying implementation.  When it calls Accept on a node, it will pass the
		// MyVisitor instance passed to it in the SetImpl call to return control flow to MyVisitor.
		*visitor.Copying
		... [ MyVisitor members ] ...
	}

	func NewAnnotationAdder() *AnnotationAdder {
		cv := visitor.NewCopying()  // Create the contained copying visitor
		v := &AnnotationAdder{Copying: cv}   // Create MyVisitor and set Copying as the base
		cv.SetImpl(v)               // Instruct the copying visitor to pass MyVisitor to Node.Accept calls
		return v
	}

	// VisitCluster re-uses the previous ast.Cluster without making a copy or modification
	func (v *AnnotationAdder) VisitCluster(c []*ast.ClusterObject) {
		return c
	}

	// VisitObject removes any object with the "testing=true" annotation while omitting copying of
	// objects that do not have the annotation.
	func (v *MyVisitor) VisitObject(o *ast.Object) {
		metaObj := o.MetaObject()
		if metaObj.GetAnnotations() != nil && metaObj.GetAnnotations()["testing"] == "true" {
			return nil
		}
		return o
	}
*/
package visitor
