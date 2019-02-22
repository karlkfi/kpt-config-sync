package syntax

import (
	v1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast/node"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	"github.com/google/nomos/pkg/policyimporter/analyzer/visitor"
	"github.com/google/nomos/pkg/util/multierror"
)

// NewNamespaceKindValidator returns a Validator that ensures only the allowed set of Kinds appear
// in Namespaces.
func NewNamespaceKindValidator() *visitor.ValidatorVisitor {
	return visitor.NewTreeNodeValidator(func(n *ast.TreeNode) error {
		eb := multierror.Builder{}
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
