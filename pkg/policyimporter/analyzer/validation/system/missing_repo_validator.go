package system

import (
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	"github.com/google/nomos/pkg/policyimporter/analyzer/visitor"
)

// NewMissingRepoValidator returns a validator which fails if Root.Repo is unset.
func NewMissingRepoValidator() ast.Visitor {
	return visitor.NewRootValidator(func(g *ast.Root) error {
		if g.Repo == nil {
			return vet.MissingRepoError{}
		}
		return nil
	})
}
