package hierarchical

import (
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/validation/system"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/util/repo"
	"github.com/google/nomos/pkg/validate/parsed"
)

// OldAllowedRepoVersion is the old (but still supported) Repo.Spec.Version.
const OldAllowedRepoVersion = "0.1.0"

var allowedRepoVersions = map[string]bool{
	repo.CurrentVersion:   true,
	OldAllowedRepoVersion: true,
}

// RepoVersionValidator returns a visitor that ensures any Repo objects in
// system/ have the correct version.
func RepoVersionValidator() parsed.ValidatorFunc {
	f := parsed.PerObjectVisitor(func(obj ast.FileObject) status.Error {
		switch repoObj := obj.Object.(type) {
		case *v1.Repo:
			if version := repoObj.Spec.Version; !allowedRepoVersions[version] {
				return system.UnsupportedRepoSpecVersion(obj, version)
			}
		}
		return nil
	})
	return parsed.ValidateSystemObjects(f)
}

// TODO(b/178219594): Move UnsupportedRepoSpecVersion error here.
