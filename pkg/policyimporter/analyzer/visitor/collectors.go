package visitor

import (
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
)

// nodeCollector collects all nodes in the policy hierarchy into the passed node list.
type nodeCollector struct {
	*Base
	nodes *[]*ast.TreeNode
}

// newNodeCollector initializes a nodeCollector which appends all nodes in the hierarchy to the
// passed list of TreeNodes.
func newNodeCollector(nodes *[]*ast.TreeNode) ast.Visitor {
	v := &nodeCollector{Base: NewBase(), nodes: nodes}
	v.SetImpl(v)
	return v
}

// VisitTreeNode implements ast.Visitor.
func (nc *nodeCollector) VisitTreeNode(node *ast.TreeNode) *ast.TreeNode {
	*nc.nodes = append(*nc.nodes, node)
	return nc.Base.VisitTreeNode(node)
}
