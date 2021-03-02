package validate

import (
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/validation/system"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/util/repo"
	"github.com/google/nomos/pkg/validate/objects"
)

// OldAllowedRepoVersion is the old (but still supported) Repo.Spec.Version.
const OldAllowedRepoVersion = "0.1.0"

var allowedRepoVersions = map[string]bool{
	repo.CurrentVersion:   true,
	OldAllowedRepoVersion: true,
}

// Repo verifies that there is exactly one Repo object and that it has the
// correct version.
func Repo(objs *objects.Raw) status.MultiError {
	var found []id.Resource
	for _, obj := range objs.Objects {
		if obj.GroupVersionKind().GroupKind() == kinds.Repo().GroupKind() {
			found = append(found, obj)
		}
	}

	if len(found) == 0 {
		return system.MissingRepoError()
	}
	if len(found) > 1 {
		return status.MultipleSingletonsError(found...)
	}

	obj := found[0].(ast.FileObject)
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
