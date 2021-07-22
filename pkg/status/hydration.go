package status

// HydrationErrorCode is the error code for a status Error related to the hydration process.
const HydrationErrorCode = "2015"

// HydrationError is an ErrorBuilder for errors related to the hydration process.
var HydrationError = NewErrorBuilder(HydrationErrorCode)

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
