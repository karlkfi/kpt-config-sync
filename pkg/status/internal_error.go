package status

// InternalErrorCode is the error code for Internal.
const InternalErrorCode = "1000"

// InternalError represents conditions that should ever happen, but that we
// check for so that we can control how the program terminates when these
// unexpected situations occur.
//
// These errors specifically happen when the code has a bug - as long as
// objects are being used as their contracts require, and as long as they
// follow their contracts, it should not be possible to trigger these.
var InternalError = wrap(NewErrorBuilder(InternalErrorCode), "internal error")

// InternalErrorf mimics the old InternalErrorf. Use InternalError.Errorf instead.
var InternalErrorf = InternalError.Errorf

// InternalWrap mimics the old InternalWrap. Use InternalError instead.
var InternalWrap = InternalError

// InternalWrapf mimics the old InternalWrapf. Use InternalError.Wrapf instead.
var InternalWrapf = InternalError.Wrapf
