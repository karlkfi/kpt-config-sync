package status

// UndocumentedErrorCode is the error code for Undocumented.
const UndocumentedErrorCode = "9999"

var undocumentedError = NewErrorBuilder(UndocumentedErrorCode)

// UndocumentedWrapf wraps an undocumented error with a formatted message.
func UndocumentedWrapf(err error, format string, a ...interface{}) Error {
	return undocumentedError.Sprintf(format, a...).Wrap(err).Build()
}

// UndocumentedErrorf returns a Undocumented with the string representation of the passed object.
func UndocumentedErrorf(format string, a ...interface{}) Error {
	return undocumentedError.Sprintf(format, a...).Build()
}

// UndocumentedError returns a Undocumented with the string representation of the passed object.
func UndocumentedError(message string) Error {
	return undocumentedError.Sprint(message).Build()
}

func undocumented(err error) Error {
	return undocumentedError.Wrap(err).Build()
}
