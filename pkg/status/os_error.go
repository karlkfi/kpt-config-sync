package status

// OSErrorCode is the error code for a status Error originating from an OS-level function call.
const OSErrorCode = "2003"

// OSWrap returns an Error wrapping an OS-level error.
var OSWrap = wrap(NewErrorBuilder(OSErrorCode), "Operating System error")
