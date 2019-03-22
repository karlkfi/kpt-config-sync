package syntax

import (
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/ast/node"
	"github.com/google/nomos/pkg/importer/analyzer/vet"
	"github.com/google/nomos/pkg/importer/analyzer/visitor"
	"github.com/google/nomos/pkg/status"
)

// NewNamespaceKindValidator returns a Validator that ensures only the allowed set of Kinds appear
// in Namespaces.
func NewNamespaceKindValidator() *visitor.ValidatorVisitor {
	return visitor.NewTreeNodeValidator(func(n *ast.TreeNode) *status.MultiError {
		eb := status.ErrorBuilder{}
		if n.Type == node.Namespace {
			for _, object := range n.Objects {
				switch object.Object.(type) {
				case *v1.NamespaceSelector:
					eb.Add(vet.IllegalKindInNamespacesError{Resource: object})
				}
			}
		}
		return eb.Build()
	})
}
