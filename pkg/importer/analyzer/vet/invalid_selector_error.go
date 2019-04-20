package vet

import (
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/status"
	"github.com/pkg/errors"
)

// InvalidSelectorErrorCode is the error code for InvalidSelectorError
const InvalidSelectorErrorCode = "1014" // TODO: Must refactor to use properly

func init() {
	status.Register(InvalidSelectorErrorCode, InvalidSelectorError{
		Name:  "name",
		Cause: errors.New("problem with selector"),
	})
}

// InvalidSelectorError is a validation error.
type InvalidSelectorError struct {
	Name  string
	Cause error
}

// Error implements error.
func (e InvalidSelectorError) Error() string {
	return status.Format(e, errors.Wrapf(e.Cause, "Selector for %q has validation errors that must be corrected", e.Name).Error())
}

// Code implements Error
func (e InvalidSelectorError) Code() string { return InvalidSelectorErrorCode }

// ToCME implements ToCMEr.
func (e InvalidSelectorError) ToCME() v1.ConfigManagementError {
	return status.FromError(e)
}
