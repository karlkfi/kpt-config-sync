package sync

import (
	"github.com/google/nomos/pkg/policyimporter/analyzer/validation/validator"
	"github.com/google/nomos/pkg/policyimporter/analyzer/veterrors"
	"github.com/google/nomos/pkg/policyimporter/id"
	"github.com/google/nomos/pkg/util/multierror"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// VersionValidatorFactory ensures duplicate Sync definitions for Group/Kind pairs do not exist in a
// to-be-provided list of []ast.FileObject.
type VersionValidatorFactory struct{}

// New creates a ValidatorFactory that adds validation errors for Group/Kind pairs with
// multiple definitions.
func (v VersionValidatorFactory) New(syncs []FileSync) VersionValidator {
	return VersionValidator{syncs: syncs}
}

// VersionValidator validates that the provided Syncs do not have duplicate Group/Kind definitions.
type VersionValidator struct {
	syncs []FileSync
}

var _ validator.Validator = VersionValidator{}

// Validate adds errors for each Group/Kind with multiple declarations.
func (v VersionValidator) Validate(errorBuilder *multierror.Builder) {
	syncKinds := make(map[schema.GroupKind][]id.Sync)
	for _, sync := range v.syncs {
		for _, k := range sync.flatten() {
			gk := k.GroupVersionKind().GroupKind()
			syncKinds[gk] = append(syncKinds[gk], sync)
		}
	}

	for _, duplicates := range syncKinds {
		if len(duplicates) > 1 {
			errorBuilder.Add(veterrors.DuplicateSyncGroupKindError{Duplicates: duplicates})
		}
	}
}
