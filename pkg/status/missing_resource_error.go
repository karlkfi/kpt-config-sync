package status

import (
	"github.com/google/nomos/pkg/importer/id"
)

// MissingResourceErrorCode is the error code for a MissingResourceError.
const MissingResourceErrorCode = "2011"

func init() {
	// TODO: add a way to generate valid error without dependency cycle.
	//status.AddExamples(MissingResourceErrorCode, MissingResourceError{})
}

var missingResourceError = NewErrorBuilder(MissingResourceErrorCode)

// MissingResourceWrap returns a MissingResourceError wrapping the given error and Resources.
func MissingResourceWrap(err error, msg string, resources ...id.Resource) Error {
	return missingResourceError.WithResources(resources...).Wrapf(err, "%s: expected resources were not found:", msg)
}
