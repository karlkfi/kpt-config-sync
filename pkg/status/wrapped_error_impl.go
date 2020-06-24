package status

import (
	"strings"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
)

type wrappedErrorImpl struct {
	underlying Error
	wrapped    error
}

var _ Error = wrappedErrorImpl{}

// Error implements error.
func (w wrappedErrorImpl) Error() string {
	return format(w)
}

// Is implements Error.
func (w wrappedErrorImpl) Is(target error) bool {
	return w.underlying.Is(target)
}

// Code implements Error.
func (w wrappedErrorImpl) Code() string {
	return w.underlying.Code()
}

// Body implements Error.
func (w wrappedErrorImpl) Body() string {
	var sb strings.Builder
	body := w.underlying.Body()
	wrapped := w.wrapped.Error()
	sb.WriteString(w.underlying.Body())
	if body != "" && wrapped != "" {
		sb.WriteString(": ")
	}
	sb.WriteString(w.wrapped.Error())
	return sb.String()
}

// Errors implements MultiError.
func (w wrappedErrorImpl) Errors() []Error {
	return []Error{w}
}

// ToCME implements Error.
func (w wrappedErrorImpl) ToCME() v1.ConfigManagementError {
	return fromError(w)
}

// Cause implements causer
func (w wrappedErrorImpl) Cause() error {
	return w.wrapped
}
