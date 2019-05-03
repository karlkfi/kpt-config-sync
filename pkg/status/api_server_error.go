package status

import (
	"github.com/pkg/errors"
)

// APIServerErrorCode is the error code for a status Error originating from the kubernetes API server.
const APIServerErrorCode = "2002"

var apiServerWrap = NewErrorBuilder(APIServerErrorCode, "%s").Wrapper()

// APIServerWrapf returns an Error wrapping a kubernetes API server error.
func APIServerWrapf(err error, format string, args ...interface{}) Error {
	return apiServerWrap(errors.Wrapf(err, format, args...))
}
