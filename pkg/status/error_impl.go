package status

import (
	"github.com/google/nomos/pkg/api/configmanagement/v1"
)

// baseErrorImpl represents a root error around which more complex errors are built.
type baseErrorImpl struct {
	code string
}

var _ Error = baseErrorImpl{}

// Error implements error.
func (e baseErrorImpl) Error() string {
	return format(e)
}

// Code implements Error.
func (e baseErrorImpl) Code() string {
	return e.code
}

// Body implements Error.
func (e baseErrorImpl) Body() string {
	return ""
}

// Errors implements MultiError.
func (e baseErrorImpl) Errors() []Error {
	return []Error{e}
}

// ToCME implements Error.
func (e baseErrorImpl) ToCME() v1.ConfigManagementError {
	return fromError(e)
}

// Cause implements causer.
func (e baseErrorImpl) Cause() error {
	return nil
}
