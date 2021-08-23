package status

import (
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// EncodeDeclaredFieldErrorCode is the error code for errors that
// happen when encoding the declared fields.
const EncodeDeclaredFieldErrorCode = "1067"

var encodeDeclaredFieldError = NewErrorBuilder(EncodeDeclaredFieldErrorCode)

// EncodeDeclaredFieldError reports that an error happens when
// encoding the declared fields for an object.
func EncodeDeclaredFieldError(resource client.Object, err error) Error {
	return encodeDeclaredFieldError.Wrap(err).
		Sprintf("failed to encode declared fields").
		BuildWithResources(resource)
}
