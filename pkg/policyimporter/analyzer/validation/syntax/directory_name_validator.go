package syntax

import (
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	"github.com/google/nomos/pkg/policyimporter/analyzer/visitor"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/util/namespaceutil"
)

// NewDirectoryNameValidator validates that directory names are valid and not reserved.
func NewDirectoryNameValidator() *visitor.ValidatorVisitor {
	return visitor.NewTreeNodeValidator(
		func(n *ast.TreeNode) *status.MultiError {
			name := n.Base()
			if namespaceutil.IsInvalid(name) {
				return status.From(vet.InvalidDirectoryNameError{Dir: n.Path})
			} else if namespaceutil.IsReserved(name) {
				return status.From(vet.ReservedDirectoryNameError{Dir: n.Path})
			}
			return nil
		})
}
