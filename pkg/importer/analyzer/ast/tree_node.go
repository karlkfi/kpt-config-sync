package ast

import (
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/ast/node"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/importer/id"
)

// TreeNode is analogous to a directory in the config hierarchy.
type TreeNode struct {
	// Path is the path this node has relative to a nomos Root.
	cmpath.Path

	// The type of the HierarchyNode
	Type        node.Type
	Labels      map[string]string
	Annotations map[string]string

	// Objects from the directory
	Objects []*NamespaceObject

	// Selectors is a map of name to NamespaceSelector objects found at this node.
	// One or more Objects may have an annotation referring to these NamespaceSelectors by name.
	Selectors map[string]*v1.NamespaceSelector

	// Extension holds visitor specific data.
	Data *Extension

	// children of the directory
	Children []*TreeNode
}

var _ id.TreeNode = &TreeNode{}

// Accept invokes VisitTreeNode on the visitor.
func (n *TreeNode) Accept(visitor Visitor) *TreeNode {
	if n == nil {
		return nil
	}
	return visitor.VisitTreeNode(n)
}

// PartialCopy makes an almost shallow copy of n.  An "almost shallow" copy of
// TreeNode make shallow copies of Children and members that are likely
// immutable.  A  deep copy is made of mutable members like Labels and
// Annotations.
func (n *TreeNode) PartialCopy() *TreeNode {
	nn := *n
	copyMapInto(n.Annotations, &nn.Annotations)
	copyMapInto(n.Labels, &nn.Labels)
	// Not sure if Selectors should be copied the same way.
	return &nn
}

// Name returns the name of the lowest-level directory in this node's path.
func (n *TreeNode) Name() string {
	return n.Base()
}

func copyMapInto(from map[string]string, to *map[string]string) {
	if from == nil {
		return
	}
	*to = make(map[string]string)
	for k, v := range from {
		(*to)[k] = v
	}
}

// GetAnnotations returns the annotations from n.  They are mutable if not nil.
func (n *TreeNode) GetAnnotations() map[string]string {
	return n.Annotations
}

// SetAnnotations replaces the annotations on the tree node with the supplied ones.
func (n *TreeNode) SetAnnotations(a map[string]string) {
	n.Annotations = a
}
