package status

// EmptySourceErrorCode is the error code for an EmptySourceError.
const EmptySourceErrorCode = "2006"

// EmptySourceErrorBuilder is an ErrorBuilder for errors related to the repo's source of truth.
var EmptySourceErrorBuilder = NewErrorBuilder(EmptySourceErrorCode)

// EmptySourceError returns an EmptySourceError when the specified number of resources would have be deleted.
func EmptySourceError(current int, resourceType string) Error {
	return EmptySourceErrorBuilder.
		Sprintf("mounted git repo appears to contain no managed %s, which would delete %d existing %s from the cluster", resourceType, current, resourceType).
		Build()
}
