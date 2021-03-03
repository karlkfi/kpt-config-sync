package fake

import (
	"fmt"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	"github.com/google/nomos/pkg/status"
)

// Errors returns a MultiError consisting of fake errors. For use in unit tests
// where multiple errors are expected to be returned.
//
// In all cases where a single error is expected, it is safe to use fake.Error
// instead.
func Errors(codes ...string) status.MultiError {
	var result status.MultiError
	for i, code := range codes {
		result = status.Append(result, fakeError{id: i + 1, code: code})
	}
	return result
}

// Error returns a fake error for use in tests which matches errors with the
// specified KNV code. This is preferable to requiring test authors to specify
// fields they don't really care about.
func Error(code string) status.Error {
	return fakeError{id: 1, code: code}
}

type fakeError struct {
	id   int
	code string
}

// Cause implements status.Error.
func (f fakeError) Cause() error {
	return nil
}

// Cause implements status.Error.
func (f fakeError) Error() string {
	return fmt.Sprintf("KNV%s fake error %d", f.code, f.id)
}

// Errors implements status.Error.
func (f fakeError) Errors() []status.Error {
	return []status.Error{f}
}

// ToCME implements status.Error.
func (f fakeError) ToCME() v1.ConfigManagementError {
	return v1.ConfigManagementError{
		Code:         f.code,
		ErrorMessage: fmt.Sprintf("fake error %d", f.id),
	}
}

// ToCSE implements status.Error.
func (f fakeError) ToCSE() v1alpha1.ConfigSyncError {
	return v1alpha1.ConfigSyncError{
		Code:         f.code,
		ErrorMessage: fmt.Sprintf("fake error %d", f.id),
	}
}

// Code implements status.Error.
func (f fakeError) Code() string {
	return f.code
}

// Body implements status.Error.
func (f fakeError) Body() string {
	return f.Error()
}

// Is implements status.Error.
func (f fakeError) Is(target error) bool {
	switch err := target.(type) {
	case status.Error:
		return err.Code() == f.code
	case status.MultiError:
		return len(err.Errors()) == 1 && err.Errors()[0].Code() == f.code
	default:
		return false
	}
}
