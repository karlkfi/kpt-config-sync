package status

import (
	"github.com/pkg/errors"
)

// UndocumentedErrorCode is the error code for Undocumented.
const UndocumentedErrorCode = "9999"

// Undocumented errors represent error conditions which we should document but have not yet.
// These should be avoided in changes that are adding a new error condition. This is purely for
// cleanup of existing code.
type Undocumented struct {
	err error
}

// Error implements error
func (i Undocumented) Error() string {
	s := Format(i, "error: %s", i.err.Error())
	return s
}

// Code implements Error
func (i Undocumented) Code() string {
	return UndocumentedErrorCode
}

// UndocumentedError returns a Undocumented with the string representation of the passed object.
func UndocumentedError(message string) Error {
	return Undocumented{err: errors.New(message)}
}

// UndocumentedErrorf returns an Undocumented with a formatted message.
func UndocumentedErrorf(format string, args ...interface{}) Error {
	return Undocumented{err: errors.Errorf(format, args...)}
}

// UndocumentedWrapf returns an Undocumented wrapping an error with a formatted message.
func UndocumentedWrapf(err error, format string, args ...interface{}) Error {
	if err == nil {
		return nil
	}
	return Undocumented{err: errors.Wrapf(err, format, args...)}
}
