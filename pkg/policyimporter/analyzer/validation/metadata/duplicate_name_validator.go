package metadata

import (
	"sort"
	"strings"

	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/policyimporter/analyzer/validation/validator"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	"github.com/google/nomos/pkg/policyimporter/id"
	"github.com/google/nomos/pkg/util/multierror"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// DuplicateNameValidatorFactory ensures passed ResourceMetas do not have name conflicts.
//
// Specifically no two Resources share the same metadata.name if they are the same GroupKind and either
// 1) they are in the same directory, or
// 2) one is in a parent directory of the other.
type DuplicateNameValidatorFactory struct{}

// New returns a DuplicateNameValidator on a specific set of ResourceMetas.
func (v DuplicateNameValidatorFactory) New(metas []ResourceMeta) validator.Validator {
	return DuplicateNameValidator{metas: metas}
}

// DuplicateNameValidator applies name collision validation to the set of ResourceMetas it contains.
type DuplicateNameValidator struct {
	metas []ResourceMeta
}

var _ validator.Validator = DuplicateNameValidator{}

// Validate adds errors to the errorBuilder for each name collision.
func (v DuplicateNameValidator) Validate(eb *multierror.Builder) {
	metasByGroupKinds := make(map[schema.GroupKind][]id.Resource)
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
func validateGroupKindCollisions(metas []id.Resource, eb *multierror.Builder) {
	metasByNames := make(map[string][]id.Resource)
	for _, meta := range metas {
		name := meta.Name()
		metasByNames[name] = append(metasByNames[name], meta)
	}

	for name, metas := range metasByNames {
		validateNameCollisions(name, metas, eb)
	}
}

// validateNameCollisions assumes all metas have the same GroupKind and metadata.name
func validateNameCollisions(name string, metas []id.Resource, eb *multierror.Builder) {
	sort.Slice(metas, func(i, j int) bool {
		// Sort by source file.
		return metas[i].Dir().RelativeSlashPath() < metas[j].Dir().RelativeSlashPath()
	})

	for i := 0; i < len(metas); {
		dir := metas[i].Dir()
		var duplicates []id.Resource

		for j := i + 1; j < len(metas); j++ {
			if strings.HasPrefix(metas[j].RelativeSlashPath(), dir.RelativeSlashPath()) {
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
			eb.Add(vet.MetadataNameCollisionError{Name: name, Duplicates: append(duplicates, metas[i])})
		}

		// Recall that len(duplicates) is 0 if there are no duplicates.
		// There's no need to have multiple errors when more than two objects collide.
		i += 1 + len(duplicates)
	}
}

// ResourceMeta provides a Resource's identifier and its metadata.
type ResourceMeta interface {
	id.Resource
	MetaObject() metav1.Object
}
