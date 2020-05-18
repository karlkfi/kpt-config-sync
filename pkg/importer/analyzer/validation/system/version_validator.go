package system

import (
	"github.com/google/nomos/pkg/api/configmanagement"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/visitor"
	"github.com/google/nomos/pkg/importer/id"
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
func NewRepoVersionValidator() ast.Visitor {
	return visitor.NewSystemObjectValidator(func(o *ast.SystemObject) status.MultiError {
		switch repoObj := o.Object.(type) {
		case *v1.Repo:
			if version := repoObj.Spec.Version; !allowedRepoVersions[version] {
				return UnsupportedRepoSpecVersion(o, version)
			}
		}
		return nil
	})
}

// UnsupportedRepoSpecVersionCode is the error code for UnsupportedRepoSpecVersion
const UnsupportedRepoSpecVersionCode = "1027"

var unsupportedRepoSpecVersion = status.NewErrorBuilder(UnsupportedRepoSpecVersionCode)

// UnsupportedRepoSpecVersion reports that the repo version is not supported.
func UnsupportedRepoSpecVersion(resource id.Resource, version string) status.Error {
	return unsupportedRepoSpecVersion.
		Sprintf(`This version of %s supports repository version %q, but this repository
declares a Repo object with spec.version: %q. Refer to the release notes at
https://cloud.google.com/anthos-config-management/docs/release-notes for
instructions on upgrading your repository.`,
			configmanagement.ProductName, repo.CurrentVersion, version).
		BuildWithResources(resource)
}
