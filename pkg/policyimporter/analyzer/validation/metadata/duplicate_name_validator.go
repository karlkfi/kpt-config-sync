package metadata

import (
	"path"
	"sort"
	"strings"

	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/policyimporter/analyzer/veterrors"
	"github.com/google/nomos/pkg/util/multierror"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// DuplicateNameValidatorFactory ensures passed ResourceMetas do not have name conflicts.
//
// Specifically no two Resources share the same metadata.name if they are the same GroupKind and either
// 1) they are in the same directory, or
// 2) one is in a parent directory of the other.
type DuplicateNameValidatorFactory struct{}

// New returns a DuplicateNameValidator on a specific set of ResourceMetas.
func (v DuplicateNameValidatorFactory) New(metas []ResourceMeta) DuplicateNameValidator {
	return DuplicateNameValidator{metas: metas}
}

// DuplicateNameValidator applies name collision validation to the set of ResourceMetas it contains.
type DuplicateNameValidator struct {
	metas []ResourceMeta
}

// Validate adds errors to the errorBuilder for each name collision.
func (v DuplicateNameValidator) Validate(eb *multierror.Builder) {
	metasByGroupKinds := make(map[schema.GroupKind][]veterrors.ResourceID)
	for i, meta := range v.metas {
		gvk := meta.GroupVersionKind()
		if gvk == kinds.ResourceQuota() {
			// ResourceQuota is exempt from these checks.
			continue
		}
		gk := gvk.GroupKind()
		metasByGroupKinds[gk] = append(metasByGroupKinds[gk], v.metas[i])
	}

	for _, m := range metasByGroupKinds {
		validateGroupKindCollisions(m, eb)
	}
}

// validateGroupKindCollisions assumes all metas have the same GroupKind
func validateGroupKindCollisions(metas []veterrors.ResourceID, eb *multierror.Builder) {
	metasByNames := make(map[string][]veterrors.ResourceID)
	for _, meta := range metas {
		name := meta.Name()
		metasByNames[name] = append(metasByNames[name], meta)
	}

	for name, metas := range metasByNames {
		validateNameCollisions(name, metas, eb)
	}
}

// validateNameCollisions assumes all metas have the same GroupKind and metadata.name
func validateNameCollisions(name string, metas []veterrors.ResourceID, eb *multierror.Builder) {
	sort.Slice(metas, func(i, j int) bool {
		// Sort by source file.
		return path.Dir(metas[i].Source()) < path.Dir(metas[j].Source())
	})

	for i := 0; i < len(metas); {
		dir := path.Dir(metas[i].Source())
		var duplicates []veterrors.ResourceID

		for j := i + 1; j < len(metas); j++ {
			if strings.HasPrefix(metas[j].Source(), dir) {
				// Pick up duplicates in the same directory and child directories.
				duplicates = append(duplicates, metas[j])
			} else {
				// Since objects are sorted by paths, this guarantees that objects within a directory and
				// its subdirectories will be contiguous. We can exit this inner loop at the first
				// non-matching source path.
				break
			}
		}

		if duplicates != nil {
			eb.Add(veterrors.MetadataNameCollisionError{Name: name, Duplicates: append(duplicates, metas[i])})
		}

		// Recall that len(duplicates) is 0 if there are no duplicates.
		// There's no need to have multiple errors when more than two objects collide.
		i += 1 + len(duplicates)
	}
}
