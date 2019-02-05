package validation

import (
	"github.com/google/nomos/bespin/pkg/kinds"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	"github.com/google/nomos/pkg/policyimporter/analyzer/visitor"
)

// NewUniqueIAMValidator returns a UniqueIAMValidator.
func NewUniqueIAMValidator() ast.Visitor {
	return visitor.NewTreeNodeValidator(validateUniqueIAM)
}

// validateUniqueIAM returns an error if the node has multiple IAMPolicies.
func validateUniqueIAM(n *ast.TreeNode) error {
	var iams []*ast.NamespaceObject

	for _, object := range n.Objects {
		if object.GroupVersionKind().GroupKind() == kinds.IAMPolicy() {
			iams = append(iams, object)
		}
	}

	if len(iams) > 1 {
		return vet.UndocumentedErrorf(
			"Illegal duplicate IAM policies in %q", n.Relative.RelativeSlashPath())
	}
	return nil
}
