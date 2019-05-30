package vet

import (
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/status"
)

// InvalidCRDNameErrorCode is the error code for InvalidCRDNameError.
const InvalidCRDNameErrorCode = "1048"

func init() {
	status.AddExamples(InvalidCRDNameErrorCode, InvalidCRDNameError(
		customResourceDefinition(),
	))
}

var invalidCRDNameError = status.NewErrorBuilder(InvalidCRDNameErrorCode)

// InvalidCRDNameError reports a CRD with an invalid name in the repo.
func InvalidCRDNameError(resource id.Resource) status.Error {
	return invalidCRDNameError.WithResources(resource).Errorf(
		"The CustomResourceDefinition has an invalid name. To fix, change the name to `spec.names.plural+\".\"+spec.group`.")
}
