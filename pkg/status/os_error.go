package status

// OSErrorCode is the error code for a status Error originating from an OS-level function call.
const OSErrorCode = "2003"

func init() {
	Register(OSErrorCode, osError{})
}

// osError results from an OS-level function call (eg fetching the current user) that fails.
type osError struct {
	err error
}

var _ Error = &osError{}

// Error implements Error.
func (p osError) Error() string {
	return Format(p, "Operating System error: %s", p.err.Error())
}

// Code implements Error.
func (p osError) Code() string {
	return OSErrorCode
}

// OSWrapf returns an Error wrapping an OS-level error.
func OSWrapf(err error) Error {
	if err == nil {
		return nil
	}
	return osError{err: err}
}
