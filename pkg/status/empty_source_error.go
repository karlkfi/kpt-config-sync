package status

// EmptySourceErrorCode is the error code for an emptySourceError.
const EmptySourceErrorCode = "2006"

// emptySourceError is an ErrorBuilder for errors related to the repo's source of truth.
var emptySourceError = NewErrorBuilder(EmptySourceErrorCode)

// EmptySourceError returns an emptySourceError when the specified number of resources would have be deleted.
func EmptySourceError(current int, resourceType string) Error {
	return emptySourceError.
		Sprintf("mounted git repo appears to contain no managed %s, which would delete %d existing %s from the cluster", resourceType, current, resourceType).
		Build()
}
