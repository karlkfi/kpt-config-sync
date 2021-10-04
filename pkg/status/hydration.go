package status

// InternalHydrationErrorCode is the error code for an internal Error related to the hydration process.
const InternalHydrationErrorCode = "2015"

// ActionableHydrationErrorCode is the error code for a user actionable Error related to the hydration process.
const ActionableHydrationErrorCode = "1068"

// InternalHydrationError is an ErrorBuilder for internal errors related to the hydration process.
var InternalHydrationError = NewErrorBuilder(InternalHydrationErrorCode)
