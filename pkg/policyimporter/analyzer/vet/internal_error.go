package vet

import (
	"github.com/google/nomos/pkg/status"
	"github.com/pkg/errors"
)

// InternalErrorCode is the error code for Internal.
const InternalErrorCode = "1000"

func init() {
	status.Register(InternalErrorCode, Internal{})
}

// Internal errors represent conditions that should ever happen, but that we check for so that
// we can control how the program terminates when these unexpected situations occur.
//
// These errors specifically happen when the code has a bug - as long as objects are being used
// as their contracts require, and as long as they follow their contracts, it should not be possible
// to trigger these.
type Internal struct {
	err error
}

// Error implements error
func (i Internal) Error() string {
	return status.Format(i, "internal error: %s", i.err.Error())
}

// Code implements Error
func (i Internal) Code() string {
	return InternalErrorCode
}

// InternalError returns an Internal with the string representation of the passed object.
func InternalError(message string) status.Error {
	return Internal{err: errors.New(message)}
}

// InternalErrorf returns an Internal with a formatted message.
func InternalErrorf(format string, args ...interface{}) status.Error {
	return Internal{err: errors.Errorf(format, args...)}
}

// InternalWrapf returns an Internal wrapping an error with a formatted message.
func InternalWrapf(err error, format string, args ...interface{}) status.Error {
	if err == nil {
		return nil
	}
	return Internal{err: errors.Wrapf(err, format, args...)}
}
