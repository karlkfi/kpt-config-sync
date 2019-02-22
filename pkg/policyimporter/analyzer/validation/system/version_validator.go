package system

import (
	v1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	"github.com/google/nomos/pkg/policyimporter/analyzer/visitor"
)

const allowedRepoVersion = "0.1.0"

// NewRepoVersionValidator returns a Validator that ensures any Repo objects in sytem/ have the
// correct version.
func NewRepoVersionValidator() *visitor.ValidatorVisitor {
	return visitor.NewSystemObjectValidator(func(o *ast.SystemObject) error {
		switch repo := o.Object.(type) {
		case *v1.Repo:
			if version := repo.Spec.Version; version != allowedRepoVersion {
				return vet.UnsupportedRepoSpecVersion{
					Resource: o,
					Version:  version,
				}
			}
		}
		return nil
	})
}
