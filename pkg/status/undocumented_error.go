package status

// UndocumentedErrorCode is the error code for Undocumented.
const UndocumentedErrorCode = "9999"

// UndocumentedError returns a Undocumented with the string representation of the passed object.
var UndocumentedError = NewErrorBuilder(UndocumentedErrorCode)

// UndocumentedWrapf mimics the old UndocumentedWrapf. Use UndocumentedError.Wrapf instead.
var UndocumentedWrapf = UndocumentedError.Wrapf

// UndocumentedErrorf mimics the old UndocumentedErrorf. Use UndocumentedError.Errorf instead.
var UndocumentedErrorf = UndocumentedError.Errorf
