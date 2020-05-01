package status

// UndocumentedErrorCode is the error code for Undocumented.
const UndocumentedErrorCode = "9999"

// UndocumentedErrorBuilder builds Undocumented errors.
var UndocumentedErrorBuilder = NewErrorBuilder(UndocumentedErrorCode)

// UndocumentedErrorf returns a Undocumented with the string representation of the passed object.
func UndocumentedErrorf(format string, a ...interface{}) Error {
	return UndocumentedErrorBuilder.Sprintf(format, a...).Build()
}

// UndocumentedError returns a Undocumented with the string representation of the passed object.
func UndocumentedError(message string) Error {
	return UndocumentedErrorBuilder.Sprint(message).Build()
}

func undocumented(err error) Error {
	return UndocumentedErrorBuilder.Wrap(err).Build()
}
