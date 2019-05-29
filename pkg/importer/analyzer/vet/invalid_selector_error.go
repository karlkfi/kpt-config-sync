package vet

import (
	"github.com/google/nomos/pkg/status"
	"github.com/pkg/errors"
)

// InvalidSelectorErrorCode is the error code for InvalidSelectorError
const InvalidSelectorErrorCode = "1014" // TODO: Must refactor to use properly

func init() {
	status.AddExamples(InvalidSelectorErrorCode, InvalidSelectorError("name", errors.New("problem with selector")))
}

var invalidSelectorError = status.NewErrorBuilder(InvalidSelectorErrorCode)

// InvalidSelectorError reports that a selector is invalid.
func InvalidSelectorError(name string, cause error) status.Error {
	return invalidSelectorError.Wrapf(cause, "Selector for %q has validation errors that must be corrected", name)
}
