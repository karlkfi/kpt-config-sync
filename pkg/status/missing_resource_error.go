package status

import (
	"github.com/google/nomos/pkg/importer/id"
)

// MissingResourceErrorCode is the error code for a MissingResourceError.
const MissingResourceErrorCode = "2011"

var missingResourceError = NewErrorBuilder(MissingResourceErrorCode)

// MissingResourceWrap returns a MissingResourceError wrapping the given error and Resources.
func MissingResourceWrap(err error, msg string, resources ...id.Resource) Error {
	return missingResourceError.
		Sprintf("%s: expected resources were not found:", msg).
		Wrap(err).
		BuildWithResources(resources...)
}
