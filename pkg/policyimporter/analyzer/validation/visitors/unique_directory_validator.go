package visitors

import (
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast/node"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	"github.com/google/nomos/pkg/policyimporter/analyzer/visitor"
	"github.com/google/nomos/pkg/policyimporter/filesystem/cmpath"
	"github.com/google/nomos/pkg/status"
)

// NewUniqueDirectoryValidator initializes a ValidatorVisitor that checks that
// directory names corresponding to leaf namespaces are globally unique.
func NewUniqueDirectoryValidator() ast.Visitor {
	return visitor.NewTreeNodesValidator(func(ns []*ast.TreeNode) *status.MultiError {
		eb := status.ErrorBuilder{}
		validateUniqueDirectories(ns, &eb)
		return eb.Build()
	})
}

func validateUniqueDirectories(nodes []*ast.TreeNode, eb *status.ErrorBuilder) {
	names := make(map[string][]cmpath.Path, len(nodes))
	for _, n := range nodes {
		if n.Type == node.AbstractNamespace {
			continue
		}
		// Only do this check on leaf nodes and their base names.
		name := n.Base()
		names[name] = append(names[name], n.Path)
	}

	for _, dirs := range names {
		if len(dirs) > 1 {
			eb.Add(vet.DuplicateDirectoryNameError{Duplicates: dirs})
		}
	}
}
