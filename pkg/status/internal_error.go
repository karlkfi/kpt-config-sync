package status

import (
	"fmt"

	"github.com/pkg/errors"
)

// InternalErrorCode is the error code for Internal.
const InternalErrorCode = "1000"

// internal errors represent conditions that should ever happen, but that we
// check for so that we can control how the program terminates when these
// unexpected situations occur.
//
// These errors specifically happen when the code has a bug - as long as
// objects are being used as their contracts require, and as long as they
// follow their contracts, it should not be possible to trigger these.
var internal = NewErrorBuilder(InternalErrorCode, "internal error: %s")

// InternalErrorf returns an Internal with a formatted message.
func InternalErrorf(format string, args ...interface{}) Error {
	return internal(fmt.Sprintf(format, args...))
}

// InternalError returns an Internal with the string representation of the passed object.
func InternalError(message string) Error {
	return internal(message)
}

var internalWrap = internal.Wrapper()

// InternalWrap wraps an error inside an Internal error.
func InternalWrap(err error) Error {
	return internalWrap(err)
}

// InternalWrapf returns an Internal wrapping an error with a formatted message.
func InternalWrapf(err error, format string, args ...interface{}) Error {
	return internalWrap(errors.Wrapf(err, format, args...))
}
