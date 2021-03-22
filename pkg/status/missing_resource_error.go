package status

import "sigs.k8s.io/controller-runtime/pkg/client"

// MissingResourceErrorCode is the error code for a MissingResourceError.
const MissingResourceErrorCode = "2011"

var missingResourceError = NewErrorBuilder(MissingResourceErrorCode)

// MissingResourceWrap returns a MissingResourceError wrapping the given error and Resources.
func MissingResourceWrap(err error, msg string, resources ...client.Object) Error {
	return missingResourceError.
		Sprintf("%s: expected resources were not found:", msg).
		Wrap(err).
		BuildWithResources(resources...)
}
