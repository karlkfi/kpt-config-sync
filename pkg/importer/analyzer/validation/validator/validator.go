package validator

import "github.com/google/nomos/pkg/status"

// Validator implements a single type of Validation on a Nomos repository.
type Validator interface {
	// Validate executes validation and adds all found validation errors to the passed
	// *status.ErrorBuilder.
	Validate(eb *status.ErrorBuilder)
}
