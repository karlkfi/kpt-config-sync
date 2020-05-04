package status

// InternalErrorCode is the error code for Internal.
const InternalErrorCode = "1000"

// InternalErrorBuilder allows creating complex internal errors.
var InternalErrorBuilder = NewErrorBuilder(InternalErrorCode).Sprint("internal error")

// InternalError represents conditions that should ever happen, but that we
// check for so that we can control how the program terminates when these
// unexpected situations occur.
//
// These errors specifically happen when the code has a bug - as long as
// objects are being used as their contracts require, and as long as they
// follow their contracts, it should not be possible to trigger these.
func InternalError(message string) Error {
	return InternalErrorBuilder.Sprint(message).Build()
}

// InternalErrorf returns an InternalError with a formatted message.
func InternalErrorf(format string, a ...interface{}) Error {
	return InternalErrorBuilder.Sprintf(format, a...).Build()
}

// InternalWrap wraps an error as an internal error.
func InternalWrap(err error) Error {
	return InternalErrorBuilder.Wrap(err).Build()
}
