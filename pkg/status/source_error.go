package status

// SourceErrorCode is the error code for a status Error related to the repo's source of truth.
const SourceErrorCode = "2004"

// SourceError is an ErrorBuilder for errors related to the repo's source of truth.
var SourceError = NewErrorBuilder(SourceErrorCode)
