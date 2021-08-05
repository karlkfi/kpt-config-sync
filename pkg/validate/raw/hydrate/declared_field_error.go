package hydrate

import (
	"github.com/google/nomos/pkg/status"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// EncodeDeclaredFieldErrorCode is the error code for errors that
// happen when encoding the declared fields.
const EncodeDeclaredFieldErrorCode = "1067"

var encodeDeclaredFieldError = status.NewErrorBuilder(EncodeDeclaredFieldErrorCode)

// EncodeDeclaredFieldError reports that an error happens when
// encoding the declared fields for an object.
func EncodeDeclaredFieldError(resource client.Object, err error) status.Error {
	return encodeDeclaredFieldError.Wrap(err).
		Sprintf("failed to encode declared fields").
		BuildWithResources(resource)
}
