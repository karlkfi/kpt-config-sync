package status

import (
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
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

// Is implements Error.
func (e baseErrorImpl) Is(target error) bool {
	// Two errors satisfy errors.Is() if they are both status.Error and have the
	// same KNV code.
	if se, ok := target.(Error); ok {
		return e.Code() == se.Code()
	}
	return false
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

// ToCSE implements Error.
func (e baseErrorImpl) ToCSE() v1alpha1.ConfigSyncError {
	return cseFromError(e)
}

// Cause implements causer.
func (e baseErrorImpl) Cause() error {
	return nil
}
