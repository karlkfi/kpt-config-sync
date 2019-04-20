package visitors

import (
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/ast/node"
	"github.com/google/nomos/pkg/importer/analyzer/vet"
	"github.com/google/nomos/pkg/importer/analyzer/visitor"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/status"
)

// NewUniqueDirectoryValidator initializes a ValidatorVisitor that checks that
// directory names corresponding to leaf namespaces are globally unique.
func NewUniqueDirectoryValidator() ast.Visitor {
	return visitor.NewTreeNodesValidator(func(ns []*ast.TreeNode) status.MultiError {
		return validateUniqueDirectories(ns)
	})
}

func validateUniqueDirectories(nodes []*ast.TreeNode) status.MultiError {
	names := make(map[string][]id.Path, len(nodes))
	for _, n := range nodes {
		if n.Type == node.AbstractNamespace {
			continue
		}
		// Only do this check on leaf nodes and their base names.
		name := n.Base()
		names[name] = append(names[name], n.Path)
	}

	var errs status.MultiError
	for _, dirs := range names {
		if len(dirs) > 1 {
			errs = status.Append(errs, vet.DuplicateDirectoryNameError{Duplicates: dirs})
		}
	}
	return errs
}
