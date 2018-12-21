package validator

import "github.com/google/nomos/pkg/util/multierror"

// Validator implements a single type of Validation on a Nomos repository.
type Validator interface {
	// Validate executes validation and adds all found validation errors to the passed
	// *multierror.Builder.
	Validate(eb *multierror.Builder)
}
