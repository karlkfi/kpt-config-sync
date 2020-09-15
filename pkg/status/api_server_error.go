package status

// APIServerErrorCode is the error code for a status Error originating from the kubernetes API server.
const APIServerErrorCode = "2002"

// APIServerErrorBuilder represents an error returned by the APIServer.
var APIServerErrorBuilder = NewErrorBuilder(APIServerErrorCode).Sprint("APIServer error")

// APIServerError wraps an error returned by the APIServer.
func APIServerError(err error, message string) Error {
	return APIServerErrorBuilder.Sprint(message).Wrap(err).Build()
}

// APIServerErrorf wraps an error returned by the APIServer with a formatted message.
func APIServerErrorf(err error, format string, a ...interface{}) Error {
	return APIServerErrorBuilder.Sprintf(format, a...).Wrap(err).Build()
}
