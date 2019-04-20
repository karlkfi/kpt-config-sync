package status

import (
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/pkg/errors"
)

// UndocumentedErrorCode is the error code for Undocumented.
const UndocumentedErrorCode = "9999"

func init() {
	Register(UndocumentedErrorCode, Undocumented{errors.New("some error")})
}

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
	return Undocumented{errors.New(message)}
}

// UndocumentedErrorf returns an Undocumented with a formatted message.
func UndocumentedErrorf(format string, args ...interface{}) Error {
	return Undocumented{errors.Errorf(format, args...)}
}

// UndocumentedWrapf returns an Undocumented wrapping an error with a formatted message.
func UndocumentedWrapf(err error, format string, args ...interface{}) Error {
	if err == nil {
		return nil
	}
	return Undocumented{errors.Wrapf(err, format, args...)}
}

// ToCME implements ToCMEr.
func (i Undocumented) ToCME() v1.ConfigManagementError {
	return FromError(i)
}
