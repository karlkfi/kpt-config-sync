package status

import (
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

// APIServerErrorCode is the error code for a status Error originating from the kubernetes API server.
const APIServerErrorCode = "2002"

// APIServerErrorBuilder represents an error returned by the APIServer.
var APIServerErrorBuilder = NewErrorBuilder(APIServerErrorCode).Sprint("APIServer error")

// InsufficientPermissionErrorCode is the error code when the reconciler has insufficient permissions to manage resources.
const InsufficientPermissionErrorCode = "2013"

// InsufficientPermissionErrorBuilder represents an error related to insufficient permissions returned by the APIServer.
var InsufficientPermissionErrorBuilder = NewErrorBuilder(InsufficientPermissionErrorCode).
	Sprint("Insufficient permission. To fix, make sure the reconciler has sufficient permissions.")

// APIServerError wraps an error returned by the APIServer.
func APIServerError(err error, message string) Error {
	if apierrors.IsForbidden(err) {
		return InsufficientPermissionErrorBuilder.Sprint(message).Wrap(err).Build()
	}
	return APIServerErrorBuilder.Sprint(message).Wrap(err).Build()
}

// APIServerErrorf wraps an error returned by the APIServer with a formatted message.
func APIServerErrorf(err error, format string, a ...interface{}) Error {
	if apierrors.IsForbidden(err) {
		return InsufficientPermissionErrorBuilder.Sprintf(format, a...).Wrap(err).Build()
	}
	return APIServerErrorBuilder.Sprintf(format, a...).Wrap(err).Build()
}
