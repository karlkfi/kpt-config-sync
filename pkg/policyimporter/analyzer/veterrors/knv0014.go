package veterrors

import "github.com/pkg/errors"

// InvalidSelectorError is a validation error.
type InvalidSelectorError struct {
	Name  string
	Cause error
}

// Error implements error.
func (e InvalidSelectorError) Error() string {
	return format(e, errors.Wrapf(e.Cause, "ClusterSelector %q has validation errors that must be corrected", e.Name).Error())
}

// Code implements Error
func (e InvalidSelectorError) Code() string { return InvalidSelectorErrorCode }
