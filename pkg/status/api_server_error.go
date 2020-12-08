package status

import (
	"github.com/google/nomos/pkg/importer/id"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

// APIServerErrorCode is the error code for a status Error originating from the kubernetes API server.
const APIServerErrorCode = "2002"

// apiServerErrorBuilder represents an error returned by the APIServer.
// This isn't exported to force the callers to be subject to the additional processing (e.g. detecting insufficient permissions).
var apiServerErrorBuilder = NewErrorBuilder(APIServerErrorCode).Sprint("APIServer error")

// InsufficientPermissionErrorCode is the error code when the reconciler has insufficient permissions to manage resources.
const InsufficientPermissionErrorCode = "2013"

// InsufficientPermissionErrorBuilder represents an error related to insufficient permissions returned by the APIServer.
var InsufficientPermissionErrorBuilder = NewErrorBuilder(InsufficientPermissionErrorCode).
	Sprint("Insufficient permission. To fix, make sure the reconciler has sufficient permissions.")

// APIServerError wraps an error returned by the APIServer.
func APIServerError(err error, message string, resources ...id.Resource) Error {
	var errorBuilder ErrorBuilder
	if apierrors.IsForbidden(err) {
		errorBuilder = InsufficientPermissionErrorBuilder.Sprint(message).Wrap(err)
	} else {
		errorBuilder = apiServerErrorBuilder.Sprint(message).Wrap(err)
	}
	if len(resources) == 0 {
		return errorBuilder.Build()
	}
	return errorBuilder.BuildWithResources(resources...)
}

// APIServerErrorf wraps an error returned by the APIServer with a formatted message.
func APIServerErrorf(err error, format string, a ...interface{}) Error {
	if apierrors.IsForbidden(err) {
		return InsufficientPermissionErrorBuilder.Sprintf(format, a...).Wrap(err).Build()
	}
	return apiServerErrorBuilder.Sprintf(format, a...).Wrap(err).Build()
}
