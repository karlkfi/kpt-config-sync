package status

import (
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
)

type messageErrorImpl struct {
	underlying Error
	message    string
}

var _ Error = messageErrorImpl{}

// Error implements error.
func (m messageErrorImpl) Error() string {
	return format(m)
}

// Is implements Error.
func (m messageErrorImpl) Is(target error) bool {
	return m.underlying.Is(target)
}

// Code implements Error.
func (m messageErrorImpl) Code() string {
	return m.underlying.Code()
}

// Body implements Error.
func (m messageErrorImpl) Body() string {
	return formatBody(m.message, ": ", m.underlying.Body())
}

// Errors implements MultiError.
func (m messageErrorImpl) Errors() []Error {
	return []Error{m}
}

// ToCME implements Error.
func (m messageErrorImpl) ToCME() v1.ConfigManagementError {
	return fromError(m)
}

// ToCSE implements Error.
func (m messageErrorImpl) ToCSE() v1.ConfigSyncError {
	return cseFromError(m)
}

// Cause implements causer.
func (m messageErrorImpl) Cause() error {
	return m.underlying.Cause()
}
