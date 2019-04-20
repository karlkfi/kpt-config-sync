package status

import (
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/pkg/errors"
)

// APIServerErrorCode is the error code for a status Error originating from the kubernetes API server.
const APIServerErrorCode = "2002"

func init() {
	Register(APIServerErrorCode, apiServerError{errors.New("api server error")})
}

// apiServerError results from a high level call to the API server (eg not involving a resource) that fails.
type apiServerError struct {
	err error
}

var _ Error = &apiServerError{}

// Error implements Error.
func (p apiServerError) Error() string {
	return Format(p, "K8s API server error: %s", p.err.Error())
}

// Code implements Error.
func (p apiServerError) Code() string {
	return APIServerErrorCode
}

// APIServerWrapf returns an Error wrapping a kubernetes API server error.
func APIServerWrapf(err error, format string, args ...interface{}) Error {
	if err == nil {
		return nil
	}
	return apiServerError{errors.Wrapf(err, format, args...)}
}

func (p apiServerError) ToCME() v1.ConfigManagementError {
	return FromError(p)
}
