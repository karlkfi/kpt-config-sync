package syntax

import (
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/vet"
	"github.com/google/nomos/pkg/importer/analyzer/visitor"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/util/namespaceutil"
)

// NewDirectoryNameValidator validates that directory names are valid and not reserved.
func NewDirectoryNameValidator() *visitor.ValidatorVisitor {
	return visitor.NewTreeNodeValidator(
		func(n *ast.TreeNode) status.MultiError {
			name := n.Base()
			if namespaceutil.IsInvalid(name) {
				return vet.InvalidDirectoryNameError(n.Path)
			} else if namespaceutil.IsReserved(name) {
				return vet.ReservedDirectoryNameError(n.Path)
			}
			return nil
		})
}
