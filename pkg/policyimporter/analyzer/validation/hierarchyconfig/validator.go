package hierarchyconfig

import (
	"github.com/google/nomos/pkg/policyimporter/analyzer/validation/validator"
	"github.com/google/nomos/pkg/util/multierror"
)

// ValidatorFactory is a function that returns an error if the supplied
// FileGroupKindHierarchyConfig is not valid. Validates each independently.
type ValidatorFactory struct {
	fn func(config FileGroupKindHierarchyConfig) error
}

// New returns a Validator with the set validation function on the set of passed FileHierarchyConfigs.
func (v ValidatorFactory) New(configs []FileHierarchyConfig) Validator {
	return Validator{fn: v.fn, configs: configs}
}

// Validator is a validation function to be applied to a specific set of FileHierarchyConfigs.
type Validator struct {
	fn      func(config FileGroupKindHierarchyConfig) error
	configs []FileHierarchyConfig
}

var _ validator.Validator = Validator{}

// Validate adds errors for each misconfigured Kind defined in a HierarchyConfig.
// It abstracts out the deeply-nested logic for extracting every Kind defined in every HierarchyConfig.
func (v Validator) Validate(errorBuilder *multierror.Builder) {
	for _, config := range v.configs {
		for _, k := range config.flatten() {
			errorBuilder.Add(v.fn(k))
		}
	}
}
