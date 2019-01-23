package validation

import (
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	"github.com/google/nomos/pkg/policyimporter/analyzer/visitor"
	"github.com/google/nomos/pkg/util/multierror"
)

const maxHierarchyDepth = 4

// MaxDepthValidator is an ast.Visitor which validates that no node in hierarchy/ has a depth greater
// than maxHierarchyDepth.
type MaxDepthValidator struct {
	*visitor.Base
	errors multierror.Builder
}

var _ ast.Visitor = &MaxDepthValidator{}

// NewMaxDepthValidator returns an initialized MaxDepthValidator.
func NewMaxDepthValidator() *MaxDepthValidator {
	v := &MaxDepthValidator{Base: visitor.NewBase()}
	v.SetImpl(v)
	return v
}

// Error implements ast.Visitor.
func (v *MaxDepthValidator) Error() error {
	return v.errors.Build()
}

// VisitTreeNode adds an error if the depth of the node is greater than maxHierarchyDepth.
func (v *MaxDepthValidator) VisitTreeNode(n *ast.TreeNode) *ast.TreeNode {
	// Splits the path from repository root into distinct elements to determine the depth of this node.
	relativePath := n.Relative.Split()
	// Subtracts one since the top level "hierarchy/" doesn't count as a level of depth.
	depth := len(relativePath) - 1
	if depth > maxHierarchyDepth {
		v.errors.Add(vet.UndocumentedErrorf(
			"Max allowed hierarchy depth of %d violated by %q", maxHierarchyDepth, n.RelativeSlashPath()))
	}
	return n
}
