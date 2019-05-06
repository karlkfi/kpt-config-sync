package system

import (
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/vet"
	"github.com/google/nomos/pkg/importer/analyzer/visitor"
	"github.com/google/nomos/pkg/status"
)

// NewMissingRepoValidator returns a validator which fails if Root.Repo is unset.
func NewMissingRepoValidator() ast.Visitor {
	return visitor.NewRootValidator(func(g *ast.Root) status.MultiError {
		if g.Repo == nil {
			return status.From(vet.MissingRepoError())
		}
		return nil
	})
}
