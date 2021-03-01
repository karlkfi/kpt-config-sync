package validate

import (
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/validation/system"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/util/repo"
)

// OldAllowedRepoVersion is the old (but still supported) Repo.Spec.Version.
const OldAllowedRepoVersion = "0.1.0"

var allowedRepoVersions = map[string]bool{
	repo.CurrentVersion:   true,
	OldAllowedRepoVersion: true,
}

// RepoVersion verifies that the Repo object in system/ has the correct version.
func RepoVersion(obj ast.FileObject) status.Error {
	if obj.GroupVersionKind() != kinds.Repo() {
		return nil
	}
	s, err := obj.Structured()
	if err != nil {
		return err
	}
	if version := s.(*v1.Repo).Spec.Version; !allowedRepoVersions[version] {
		return system.UnsupportedRepoSpecVersion(obj, version)
	}
	return nil
}

// TODO(b/178219594): Move UnsupportedRepoSpecVersion error here.
