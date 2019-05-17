package status

import (
	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/pkg/errors"
)

type errorImpl struct {
	error error
	code  string
}

var _ Error = errorImpl{}
var _ Causer = errorImpl{}

// Error implements error.
func (e errorImpl) Error() string {
	return format(e.error, "", e.code)
}

// Cause returns the the error that caused this error
func (e errorImpl) Cause() error {
	return errors.Cause(e.error)
}

// Code implements Error.
func (e errorImpl) Code() string {
	return e.code
}

// ToCME implements Error.
func (e errorImpl) ToCME() v1.ConfigManagementError {
	return FromError(e)
}

// ErrorBuilder is a functor that returns an Error if supplied an error.
//
// Libraries should generally not directly expose ErrorBuilders, but keep them package private and
// provide functions that tell callers the correct number and position of formatting arguments.
// The main exception is general-purpose errors like InternalError.
type ErrorBuilder func(err error) Error

// NewErrorBuilder returns a functor that can be used to generate errors. Registers this
// call with the passed unique code. Panics if there is an error code collision.
func NewErrorBuilder(code string) ErrorBuilder {
	register(code)
	return func(err error) Error {
		if err == nil {
			return nil
		}
		return errorImpl{
			error: err,
			code:  code,
		}
	}
}

// wrap wraps the ErrorBuilder with a static message.
func wrap(ew ErrorBuilder, message string) ErrorBuilder {
	return func(err error) Error {
		return ew(errors.Wrap(err, message))
	}
}

// Wrap wraps err with a static message.
func (eb ErrorBuilder) Wrap(err error, message string) Error {
	return eb(errors.Wrap(err, message))
}

// Wrapf wraps err with a formatted message.
func (eb ErrorBuilder) Wrapf(err error, format string, a ...interface{}) Error {
	return eb(errors.Wrapf(err, format, a...))
}

// Errorf instantiates an Error with the formatted message.
func (eb ErrorBuilder) Errorf(format string, a ...interface{}) Error {
	return eb(errors.Errorf(format, a...))
}

// New instantiates an Error with the passed static message.
func (eb ErrorBuilder) New(message string) Error {
	return eb(errors.New(message))
}

// WithPaths adds the passed paths to the error in a structured way.
func (eb ErrorBuilder) WithPaths(paths ...id.Path) ErrorBuilder {
	return func(err error) Error {
		return pathErrorImpl{
			errorImpl: eb(err).(errorImpl),
			paths:     paths,
		}
	}
}

// WithResources adds the passed resources to the error in a structured way.
func (eb ErrorBuilder) WithResources(resources ...id.Resource) ErrorBuilder {
	return func(err error) Error {
		return resourceErrorImpl{
			errorImpl: eb(err).(errorImpl),
			resources: resources,
		}
	}
}
