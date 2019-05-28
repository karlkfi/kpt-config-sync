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
	VisitClusterObject(o *ClusterObject) *ClusterObject
	VisitClusterRegistryObject(o *ClusterRegistryObject) *ClusterRegistryObject
	VisitSystemObject(o *SystemObject) *SystemObject
	VisitTreeNode(n *TreeNode) *TreeNode
	VisitObject(o *NamespaceObject) *NamespaceObject

	// Error allows the visitor to emit errors that may have occurred while operating.
	Error() status.MultiError

	// Fatal returns if the Visitor has determined that Parser should stop processing immediately.
	Fatal() bool

	// RequiresValidState returns whether this Visitor should run if the Parser has encountered any
	// errors whatsoever.
	RequiresValidState() bool
}
