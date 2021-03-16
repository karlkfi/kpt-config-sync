package ast

import (
	"github.com/google/nomos/pkg/importer/analyzer/ast/node"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
)

// TreeNode is analogous to a directory in the config hierarchy.
type TreeNode struct {
	// Path is the path this node has relative to a nomos Root.
	cmpath.Relative

	// The type of the HierarchyNode
	Type node.Type

	// Objects from the directory
	Objects []FileObject

	// children of the directory
	Children []*TreeNode
}

var _ id.TreeNode = &TreeNode{}

// PartialCopy makes an almost shallow copy of n.  An "almost shallow" copy of
// TreeNode make shallow copies of Children and members that are likely
// immutable.  A  deep copy is made of mutable members like Labels and
// Annotations.
func (n *TreeNode) PartialCopy() *TreeNode {
	nn := *n
	// Not sure if Selectors should be copied the same way.
	return &nn
}

// Name returns the name of the lowest-level directory in this node's path.
func (n *TreeNode) Name() string {
	return n.Base()
}

// Flatten returns the list of materialized FileObjects contained in this
// TreeNode. Specifically, it returns either
// 1) the list of Objects if this is a Namespace node, or
// 2) the concatenated list of all objects returned by calling Flatten on all of
// its children.
func (n *TreeNode) Flatten() []FileObject {
	switch n.Type {
	case node.Namespace:
		return n.flattenNamespace()
	case node.AbstractNamespace:
		return n.flattenAbstractNamespace()
	default:
		panic(status.InternalErrorf("invalid node type: %q", string(n.Type)))
	}
}

func (n *TreeNode) flattenNamespace() []FileObject {
	var result []FileObject
	for _, o := range n.Objects {
		if o.GroupVersionKind() != kinds.Namespace() {
			o.SetNamespace(n.Name())
		}
		result = append(result, o)
	}
	return result
}

func (n *TreeNode) flattenAbstractNamespace() []FileObject {
	var result []FileObject

	for _, o := range n.Objects {
		if o.GroupVersionKind() == kinds.NamespaceSelector() {
			result = append(result, o)
		}
	}
	for _, child := range n.Children {
		result = append(result, child.Flatten()...)
	}

	return result
}
