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

import "github.com/google/nomos/pkg/status"

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
	VisitRoot(g *Root) *Root
	VisitCluster(c *Cluster) *Cluster
	VisitClusterObject(o *ClusterObject) *ClusterObject
	VisitClusterRegistry(c *ClusterRegistry) *ClusterRegistry
	VisitClusterRegistryObject(o *ClusterRegistryObject) *ClusterRegistryObject
	VisitSystem(c *System) *System
	VisitSystemObject(o *SystemObject) *SystemObject
	VisitTreeNode(n *TreeNode) *TreeNode
	VisitObject(o *NamespaceObject) *NamespaceObject

	// Error allows the visitor to emit errors that may have occurred while operating.
	Error() *status.MultiError

	// Fatal returns if the Visitor has determined that Parser should stop processing immediately.
	Fatal() bool

	// RequiresValidState returns whether this Visitor should run if the Parser has encountered any
	// errors whatsoever.
	RequiresValidState() bool
}
