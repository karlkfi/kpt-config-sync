package vet

import (
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/status"
)

// ObjectParseErrorCode is the code for ObjectParseError.
const ObjectParseErrorCode = "1006"

func init() {
	status.AddExamples(ObjectParseErrorCode, ObjectParseError(
		role(),
	))
}

var objectParseError = status.NewErrorBuilder(ObjectParseErrorCode)

// ObjectParseError reports that an object of known type did not match its definition, and so it was
// read in as an *unstructured.Unstructured.
func ObjectParseError(resource id.Resource) status.Error {
	return objectParseError.WithResources(resource).Errorf(
		"The following config is not parseable as a %v:", resource.GroupVersionKind())
}
