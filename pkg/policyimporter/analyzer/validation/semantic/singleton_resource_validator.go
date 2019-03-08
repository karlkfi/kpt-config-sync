package semantic

import (
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	"github.com/google/nomos/pkg/policyimporter/analyzer/visitor"
	"github.com/google/nomos/pkg/policyimporter/id"
	"github.com/google/nomos/pkg/status"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// NewSingletonResourceValidator returns a ValidatorVisitor which ensures every TreeNode has
// at most one of the passed GroupVersionKind.
func NewSingletonResourceValidator(gvk schema.GroupVersionKind) *visitor.ValidatorVisitor {
	return visitor.NewAllNodesValidator(func(objects []ast.FileObject) *status.MultiError {
		var duplicates []id.Resource
		for _, object := range objects {
			if object.GroupVersionKind() == gvk {
				duplicates = append(duplicates, &object)
			}
		}
		if len(duplicates) > 1 {
			return status.From(vet.MultipleSingletonsError{Duplicates: duplicates})
		}
		return nil
	})
}
