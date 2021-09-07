package status

// InternalHydrationErrorCode is the error code for an internal Error related to the hydration process.
const InternalHydrationErrorCode = "2015"

// ActionableHydrationErrorCode is the error code for a user actionable Error related to the hydration process.
const ActionableHydrationErrorCode = "1068"

// InternalHydrationError is an ErrorBuilder for internal errors related to the hydration process.
var InternalHydrationError = NewErrorBuilder(InternalHydrationErrorCode)

// HydrationInProgressCode is the code for a status related to the hydration process.
// Technically, it is not an error. It indicates the configs are not available for
// the reconciler to consume (read, parse and apply).
const HydrationInProgressCode = "9997"

// hydrationInProgressBuilder is an ErrorBuilder for the HydrationInProgress status.
var hydrationInProgressBuilder = NewErrorBuilder(HydrationInProgressCode)

// HydrationInProgress returns an HydrationInProgress status with a formatted message.
func HydrationInProgress(commit string) Error {
	return hydrationInProgressBuilder.Sprintf("rendering in progress for commit %s", commit).Build()
}
