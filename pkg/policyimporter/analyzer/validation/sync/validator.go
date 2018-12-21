package sync

import (
	"github.com/google/nomos/pkg/policyimporter/analyzer/validation/validator"
	"github.com/google/nomos/pkg/util/multierror"
)

// ValidatorFactory is a function that returns an error if the supplied
// FileGroupVersionKindHierarchySync is not valid. Validates each independently.
type ValidatorFactory struct {
	fn func(sync FileGroupVersionKindHierarchySync) error
}

// nilValidatorFn is a no-op ValidatorFactory to be used when the particular Sync validator
// is unsafe or impossible to use.
var nilValidatorFactory = ValidatorFactory{
	fn: func(sync FileGroupVersionKindHierarchySync) error { return nil },
}

// New returns a ValidatorFactory with the set validation function on the set of passed objects.
func (v ValidatorFactory) New(syncs []FileSync) Validator {
	return Validator{fn: v.fn, syncs: syncs}
}

// Validator is a validation function to be applied to a specific set of syncs.
type Validator struct {
	fn    func(sync FileGroupVersionKindHierarchySync) error
	syncs []FileSync
}

var _ validator.Validator = Validator{}

// Validate adds errors for each misconfigured Kind defined in a Sync.
// It abstracts out the deeply-nested logic for extracting every Kind defined in every Sync.
func (v Validator) Validate(errorBuilder *multierror.Builder) {
	for _, sync := range v.syncs {
		for _, k := range sync.flatten() {
			errorBuilder.Add(v.fn(k))
		}
	}
}
