package system

import (
	"github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	"github.com/google/nomos/pkg/policyimporter/analyzer/visitor"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/util/repo"
)

// AllowedRepoVersion is the allowed version for Repo.Spec.Version.
const AllowedRepoVersion = repo.CurrentVersion

// NewRepoVersionValidator returns a Validator that ensures any Repo objects in sytem/ have the
// correct version.
func NewRepoVersionValidator() *visitor.ValidatorVisitor {
	return visitor.NewSystemObjectValidator(func(o *ast.SystemObject) *status.MultiError {
		switch repoObj := o.Object.(type) {
		case *v1.Repo:
			if version := repoObj.Spec.Version; version != AllowedRepoVersion {
				return status.From(vet.UnsupportedRepoSpecVersion{
					Resource: o,
					Version:  version,
				})
			}
		}
		return nil
	})
}
