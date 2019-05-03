package status

import (
	"fmt"

	"github.com/google/nomos/pkg/api/configmanagement/v1"
)

type errorImpl struct {
	error error
	code  string
}

// Error implements error.
func (e errorImpl) Error() string {
	return e.error.Error()
}

// Code implements Error.
func (e errorImpl) Code() string {
	return e.code
}

// ToCME implements Error.
func (e errorImpl) ToCME() v1.ConfigManagementError {
	return FromError(e)
}

// ErrorWrapper is a functor that returns an Error if supplied an error and formatting arguments.
type ErrorWrapper func(err error, a ...interface{}) Error

// ErrorBuilder is a functor that returns an Error if supplied formatting arguments.
type ErrorBuilder func(a ...interface{}) Error

// NewErrorBuilder returns a functor that can be used to generate errors. Registers this
// ErrorBuilder with the passed code.
//
// Callers should not directly expose ErrorBuilders, but keep them package private and provide
// functions that tell callers the correct number and position of formatting arguments.
func NewErrorBuilder(code string, format string) ErrorBuilder {
	// TODO: Allow registering examples.
	Register(code, nil)
	return func(a ...interface{}) Error {
		return errorImpl{
			error: fmt.Errorf(format, a...),
			code:  code,
		}
	}
}

// Wrapper returns the an error wrapper form of the ErrorBuilder.
//
// Returns nil if supplied a nil error.
func (eb ErrorBuilder) Wrapper() ErrorWrapper {
	return func(err error, a ...interface{}) Error {
		if err == nil {
			return nil
		}
		return eb(append([]interface{}{err.Error()}, a...))
	}
}
