package status

// APIServerErrorCode is the error code for a status Error originating from the kubernetes API server.
const APIServerErrorCode = "2002"

// APIServerError represents an error returned by the APIServer.
var APIServerError = NewErrorBuilder(APIServerErrorCode)

// APIServerWrapf mimics the old APIServerWrap. To be deprecated.
var APIServerWrapf = APIServerError.Wrapf
