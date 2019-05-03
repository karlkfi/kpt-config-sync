package status

import (
	"fmt"

	"github.com/pkg/errors"
)

// UndocumentedErrorCode is the error code for Undocumented.
const UndocumentedErrorCode = "9999"

var undocumented = NewErrorBuilder(UndocumentedErrorCode, "%s")

// UndocumentedError returns a Undocumented with the string representation of the passed object.
func UndocumentedError(message string) Error {
	return undocumented(message)
}

// UndocumentedErrorf returns an Undocumented with a formatted message.
func UndocumentedErrorf(format string, args ...interface{}) Error {
	return undocumented(fmt.Sprintf(format, args...))
}

var undocumentedWrap = undocumented.Wrapper()

// UndocumentedWrapf returns an Undocumented wrapping an error with a formatted message.
func UndocumentedWrapf(err error, format string, args ...interface{}) Error {
	return undocumentedWrap(errors.Wrapf(err, format, args...))
}
