package id

import (
	"github.com/google/nomos/pkg/status"
	"github.com/pkg/errors"
)

// MissingResourceErrorCode is the error code for a MissingResourceError.
const MissingResourceErrorCode = "2011"

func init() {
	// TODO: add a way to generate valid error without dependency cycle.
	//status.Register(MissingResourceErrorCode, MissingResourceError{})
}

// MissingResourceError reports that one or more Resources were not found by the API server.
type MissingResourceError struct {
	err       error
	resources []Resource
}

var _ ResourceError = &MissingResourceError{}

// Error implements status.Error
func (m MissingResourceError) Error() string {
	return status.Format(m, "%[1]s\nExpected resources were not found:\n%[2]s",
		m.err.Error(), FormatResources(m))
}

// Code implements status.Error
func (m MissingResourceError) Code() string {
	return MissingResourceErrorCode
}

// Resources implements ResourceError
func (m MissingResourceError) Resources() []Resource {
	return m.resources
}

// MissingResourceWrap returns a MissingResourceError wrapping the given error and Resources.
func MissingResourceWrap(err error, msg string, resources ...Resource) MissingResourceError {
	return MissingResourceError{err: errors.Wrap(err, msg), resources: resources}
}
