package status

// OSErrorCode is the error code for a status Error originating from an OS-level function call.
const OSErrorCode = "2003"

var osError = NewErrorBuilder(OSErrorCode).Sprint("operating system error")

// OSWrap returns an Error wrapping an OS-level error.
func OSWrap(err error) Error {
	return osError.Wrap(err).Build()
}
