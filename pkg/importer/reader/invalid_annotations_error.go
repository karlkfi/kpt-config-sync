package reader

import (
	"strings"

	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/status"
)

// InvalidAnnotationValueErrorCode is the error code for when a value in
// metadata.annotations is not a string.
const InvalidAnnotationValueErrorCode = "1054"

var invalidAnnotationValueErrorBase = status.NewErrorBuilder(InvalidAnnotationValueErrorCode)

// InvalidAnnotationValueError reports that an annotation value is coerced to
// a non-string type.
func InvalidAnnotationValueError(resource id.Resource, keys []string) status.ResourceError {
	return invalidAnnotationValueErrorBase.
		Sprintf("Values in metadata.annotations MUST be strings. "+
			`To fix, add quotes around the values. Non-string values for:

metadata.annotations.%s `,
			strings.Join(keys, "\nmetadata.annotations.")).
		BuildWithResources(resource)
}
