package status

import "sigs.k8s.io/controller-runtime/pkg/client"

// ObjectParseErrorCode is the code for ObjectParseError.
const ObjectParseErrorCode = "1006"

var objectParseError = NewErrorBuilder(ObjectParseErrorCode)

// ObjectParseError reports that an object of known type did not match its
// definition, and so it was read in as an *unstructured.Unstructured.
func ObjectParseError(resource client.Object, err error) Error {
	return objectParseError.Wrap(err).
		Sprintf("The following config could not be parsed as a %v", resource.GetObjectKind().GroupVersionKind()).
		BuildWithResources(resource)
}
