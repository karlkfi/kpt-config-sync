package semantic

import (
	"strings"

	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	"github.com/google/nomos/pkg/policyimporter/analyzer/visitor"
	"github.com/google/nomos/pkg/policyimporter/id"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// NewSingletonResourceValidator returns a ValidatorVisitor which ensures every TreeNode has
// at most one of the passed GroupVersionKind.
func NewSingletonResourceValidator(gvk schema.GroupVersionKind) *visitor.ValidatorVisitor {
	return visitor.NewTreeNodeValidator(func(n *ast.TreeNode) error {
		var duplicates []id.Resource
		for _, object := range n.Objects {
			if object.GroupVersionKind() == gvk {
				duplicates = append(duplicates, object)
			}
		}
		if len(duplicates) > 1 {
			switch gvk {
			case kinds.Namespace():
				return vet.MultipleNamespacesError{Duplicates: duplicates}
			default:
				resources := make([]string, len(duplicates))
				for i, duplicate := range duplicates {
					resources[i] = id.PrintResource(duplicate)
				}
				return vet.UndocumentedErrorf("At most one resource of Kind %q is allowed in each directory:\n\n%s",
					gvk.String(), strings.Join(resources, "\n\n"))
			}
		}
		return nil
	})
}
