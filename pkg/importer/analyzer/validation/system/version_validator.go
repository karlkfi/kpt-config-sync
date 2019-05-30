package system

import (
	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/vet"
	"github.com/google/nomos/pkg/importer/analyzer/visitor"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/util/repo"
)

// OldAllowedRepoVersion is the old (but still supported) Repo.Spec.Version.
const OldAllowedRepoVersion = "0.1.0"

var allowedRepoVersions = map[string]bool{
	repo.CurrentVersion:   true,
	OldAllowedRepoVersion: true,
}

// NewRepoVersionValidator returns a Validator that ensures any Repo objects in sytem/ have the
// correct version.
func NewRepoVersionValidator() *visitor.ValidatorVisitor {
	return visitor.NewSystemObjectValidator(func(o *ast.SystemObject) status.MultiError {
		switch repoObj := o.Object.(type) {
		case *v1.Repo:
			if version := repoObj.Spec.Version; !allowedRepoVersions[version] {
				return status.From(vet.UnsupportedRepoSpecVersion(
					o,
					version,
				))
			}
		}
		return nil
	})
}
