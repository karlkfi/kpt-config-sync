package validation

import (
	"github.com/google/nomos/bespin/pkg/kinds"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	"github.com/google/nomos/pkg/policyimporter/analyzer/visitor"
	"github.com/google/nomos/pkg/util/multierror"
)

const maxFolderDepth = 4

// MaxFolderDepthValidator ensures Folder directories are not stacked more than 4 times.
type MaxFolderDepthValidator struct {
	*visitor.Base
	depth  int
	errors multierror.Builder
}

var _ ast.Visitor = &MaxFolderDepthValidator{}

// NewMaxFolderDepthValidator initializes a MaxFolderDepthValidator.
func NewMaxFolderDepthValidator() ast.Visitor {
	v := &MaxFolderDepthValidator{Base: visitor.NewBase()}
	v.SetImpl(v)
	return v
}

// VisitTreeNode implements ast.Visitor.
func (v *MaxFolderDepthValidator) VisitTreeNode(n *ast.TreeNode) *ast.TreeNode {
	if isFolder(n) {
		v.depth++
	}
	if v.depth > maxFolderDepth {
		v.errors.Add(vet.UndocumentedErrorf("Max allowed hierarchy Folder depth of %d violated by %q", maxFolderDepth, n.RelativeSlashPath()))
		// No need to visit child nodes and produce an error for every child.
	} else {
		v.Base.VisitTreeNode(n)
	}
	if isFolder(n) {
		v.depth--
	}
	return n
}

// Error implements ast.Visitor.
func (v *MaxFolderDepthValidator) Error() error {
	return v.errors.Build()
}

func isFolder(n *ast.TreeNode) bool {
	for _, o := range n.Objects {
		if o.GroupVersionKind().GroupKind() == kinds.Folder() {
			return true
		}
	}
	return false
}
