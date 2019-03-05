package vet

import "github.com/pkg/errors"

// InvalidSelectorErrorCode is the error code for InvalidSelectorError
const InvalidSelectorErrorCode = "1014" // TODO: Must refactor to use properly

func init() {
	register(InvalidSelectorErrorCode, nil, "")
}

// InvalidSelectorError is a validation error.
type InvalidSelectorError struct {
	Name  string
	Cause error
}

// Error implements error.
func (e InvalidSelectorError) Error() string {
	return format(e, errors.Wrapf(e.Cause, "Label selector for %q has validation errors that must be corrected", e.Name).Error())
}

// Code implements Error
func (e InvalidSelectorError) Code() string { return InvalidSelectorErrorCode }
