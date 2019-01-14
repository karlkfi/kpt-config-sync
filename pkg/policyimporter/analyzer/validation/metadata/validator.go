package metadata

import (
	"github.com/google/nomos/pkg/policyimporter/analyzer/validation/validator"
	"github.com/google/nomos/pkg/util/multierror"
)

// ValidatorFactory is a factory that returns a validator for a given set of ResourceMeta.
type ValidatorFactory interface {
	New(metas []ResourceMeta) validator.Validator
}

// SyntaxValidatorFactory is a function that returns an error if the supplied ResourceMeta
// is not valid. Validates each ResourceMeta independently.
type SyntaxValidatorFactory struct {
	fn func(meta ResourceMeta) error
}

var _ ValidatorFactory = SyntaxValidatorFactory{}

// New returns a Validator with the set validation function on the set of passed ResourceMetas.
func (v SyntaxValidatorFactory) New(metas []ResourceMeta) validator.Validator {
	return Validator{fn: v.fn, metas: metas}
}

// Validator is a validation function to be applied to a specific set of ResourceMetas.
type Validator struct {
	fn    func(meta ResourceMeta) error
	metas []ResourceMeta
}

var _ validator.Validator = Validator{}

// Validate implements validation.Validator
func (v Validator) Validate(eb *multierror.Builder) {
	for _, meta := range v.metas {
		eb.Add(v.fn(meta))
	}
}
