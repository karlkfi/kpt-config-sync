package status

// OSErrorCode is the error code for a status Error originating from an OS-level function call.
const OSErrorCode = "2003"

var osWrapf = NewErrorBuilder(OSErrorCode, "Operating System error: %s").Wrapper()

// OSWrapf returns an Error wrapping an OS-level error.
func OSWrapf(err error) Error {
	return osWrapf(err)
}
