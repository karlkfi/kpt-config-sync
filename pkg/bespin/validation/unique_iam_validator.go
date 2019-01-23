package validation

import (
	"github.com/google/nomos/pkg/bespin/kinds"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	"github.com/google/nomos/pkg/policyimporter/analyzer/visitor"
	"github.com/google/nomos/pkg/util/multierror"
)

// UniqueIAMValidator validates that no hierarchy node may contain multiple IAMPolicies.
type UniqueIAMValidator struct {
	*visitor.Base
	errors multierror.Builder
}

var _ ast.Visitor = &UniqueIAMValidator{}

// NewUniqueIAMValidator returns a UniqueIAMValidator.
func NewUniqueIAMValidator() *UniqueIAMValidator {
	v := &UniqueIAMValidator{Base: visitor.NewBase()}
	v.SetImpl(v)
	return v
}

// Error implements ast.Visitor.
func (v *UniqueIAMValidator) Error() error {
	return v.errors.Build()
}

// VisitTreeNode adds an error for every node with multiple IAMPolicies.
func (v *UniqueIAMValidator) VisitTreeNode(n *ast.TreeNode) *ast.TreeNode {
	var iams []*ast.NamespaceObject

	for _, object := range n.Objects {
		if object.GroupVersionKind().GroupKind() == kinds.IAMPolicy() {
			iams = append(iams, object)
		}
	}

	if len(iams) > 1 {
		v.errors.Add(vet.UndocumentedErrorf(
			"Illegal duplicate IAM policies in %q", n.Relative.RelativeSlashPath()))
	}
	return n
}
