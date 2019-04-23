package status

import (
	"fmt"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/pkg/errors"
)

// InternalErrorCode is the error code for Internal.
const InternalErrorCode = "1000"

func init() {
	Register(InternalErrorCode, Internal{})
}

// Internal errors represent conditions that should ever happen, but that we
// check for so that we can control how the program terminates when these
// unexpected situations occur.
//
// These errors specifically happen when the code has a bug - as long as
// objects are being used as their contracts require, and as long as they
// follow their contracts, it should not be possible to trigger these.
type Internal struct {
	err error
}

// Error implements error
func (i Internal) Error() string {
	return Format(i, "internal error: %s", i.err.Error())
}

// Code implements Error
func (i Internal) Code() string {
	return InternalErrorCode
}

// InternalErrorf returns an Internal with a formatted message.
func InternalErrorf(format string, args ...interface{}) Error {
	return InternalError(fmt.Sprintf(format, args...))
}

// InternalError returns an Internal with the string representation of the passed object.
func InternalError(message string) Error {
	return InternalWrap(errors.New(message))
}

// InternalWrap returns an Internal wrapping an error.
func InternalWrap(err error) Error {
	if err == nil {
		return nil
	}
	return Internal{err}
}

// InternalWrapf returns an Internal wrapping an error with a formatted message.
func InternalWrapf(err error, format string, args ...interface{}) Error {
	return InternalWrap(errors.Wrapf(err, format, args...))
}

// ToCME implements ToCMEr.
func (i Internal) ToCME() v1.ConfigManagementError {
	return FromError(i)
}
