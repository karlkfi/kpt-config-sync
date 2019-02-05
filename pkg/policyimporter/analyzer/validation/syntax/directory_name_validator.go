package syntax

import (
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	"github.com/google/nomos/pkg/policyimporter/analyzer/visitor"
	"github.com/google/nomos/pkg/util/namespaceutil"
)

// NewDirectoryNameValidator validates that directory names are valid and not reserved.
func NewDirectoryNameValidator() *visitor.ValidatorVisitor {
	return visitor.NewTreeNodeValidator(
		func(n *ast.TreeNode) error {
			name := n.Base()
			if namespaceutil.IsInvalid(name) {
				return vet.InvalidDirectoryNameError{Dir: n.Relative}
			} else if namespaceutil.IsReserved(name) {
				return vet.ReservedDirectoryNameError{Dir: n.Relative}
			}
			return nil
		})
}
