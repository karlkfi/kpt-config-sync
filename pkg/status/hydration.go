package status

// InternalHydrationErrorCode is the error code for an internal Error related to the hydration process.
const InternalHydrationErrorCode = "2015"

// ActionableHydrationErrorCode is the error code for a user actionable Error related to the hydration process.
const ActionableHydrationErrorCode = "1068"

// internalHydrationErrorBuilder is an ErrorBuilder for internal errors related to the hydration process.
var internalHydrationErrorBuilder = NewErrorBuilder(InternalHydrationErrorCode)

// actionableHydrationErrorBuilder is an ErrorBuilder for user actionable errors related to the hydration process.
var actionableHydrationErrorBuilder = NewErrorBuilder(ActionableHydrationErrorCode)

// InternalHydrationError returns an internal error related to the hydration process.
func InternalHydrationError(err error, format string, a ...interface{}) Error {
	return internalHydrationErrorBuilder.Wrap(err).Sprintf(format, a...).Build()
}

// HydrationError returns a hydration error.
func HydrationError(code string, err error) Error {
	if code == ActionableHydrationErrorCode {
		return actionableHydrationErrorBuilder.Wrap(err).Build()
	}
	return internalHydrationErrorBuilder.Wrap(err).Build()
}
