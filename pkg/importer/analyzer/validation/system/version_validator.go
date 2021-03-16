package system

import (
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/util/repo"
)

// OldAllowedRepoVersion is the old (but still supported) Repo.Spec.Version.
const OldAllowedRepoVersion = "0.1.0"

// UnsupportedRepoSpecVersionCode is the error code for UnsupportedRepoSpecVersion
const UnsupportedRepoSpecVersionCode = "1027"

var unsupportedRepoSpecVersion = status.NewErrorBuilder(UnsupportedRepoSpecVersionCode)

// UnsupportedRepoSpecVersion reports that the repo version is not supported.
func UnsupportedRepoSpecVersion(resource id.Resource, version string) status.Error {
	return unsupportedRepoSpecVersion.
		Sprintf(`Unsupported Repo spec.version: %q. Must use version %q`, version, repo.CurrentVersion).
		BuildWithResources(resource)
}
