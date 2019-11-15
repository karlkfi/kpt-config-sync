package status

// APIServerErrorCode is the error code for a status Error originating from the kubernetes API server.
const APIServerErrorCode = "2002"

// APIServerError represents an error returned by the APIServer.
var apiServerError = NewErrorBuilder(APIServerErrorCode).Sprint("APIServer error")

// APIServerError wraps an error returned by the APIServer.
func APIServerError(err error, message string) Error {
	return apiServerError.Sprint(message).Wrap(err).Build()
}

// APIServerErrorf wraps an error returned by the APIServer with a formatted message.
func APIServerErrorf(err error, format string, a ...interface{}) Error {
	return apiServerError.Sprintf(format, a...).Wrap(err).Build()
}
