package visitors

import (
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	"github.com/google/nomos/pkg/policyimporter/analyzer/visitor"
	"github.com/google/nomos/pkg/policyimporter/filesystem/nomospath"
	"github.com/google/nomos/pkg/util/multierror"
)

// NewUniqueDirectoryValidator initializes a ValidatorVisitor that checks that directory names are globally
// unique.
func NewUniqueDirectoryValidator() *visitor.ValidatorVisitor {
	return visitor.NewTreeNodesValidator(func(ns []*ast.TreeNode) error {
		eb := multierror.Builder{}
		validateUniqueDirectories(ns, &eb)
		return eb.Build()
	})
}

func validateUniqueDirectories(nodes []*ast.TreeNode, eb *multierror.Builder) {
	names := make(map[string][]nomospath.Relative, len(nodes))
	for _, node := range nodes {
		name := node.Name()
		names[name] = append(names[name], node.Relative)
	}

	for _, dirs := range names {
		if len(dirs) > 1 {
			eb.Add(vet.DuplicateDirectoryNameError{Duplicates: dirs})
		}
	}
}
