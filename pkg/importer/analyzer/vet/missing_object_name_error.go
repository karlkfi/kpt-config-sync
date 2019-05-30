package vet

import (
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/status"
)

// MissingObjectNameErrorCode is the error code for MissingObjectNameError
const MissingObjectNameErrorCode = "1031"

func init() {
	r := role()
	r.MetaObject().SetName("")
	status.AddExamples(MissingObjectNameErrorCode, MissingObjectNameError(r))
}

var missingObjectNameError = status.NewErrorBuilder(MissingObjectNameErrorCode)

// MissingObjectNameError reports that an object has no name.
func MissingObjectNameError(resource id.Resource) status.Error {
	return missingObjectNameError.WithResources(resource).Errorf(
		"Configs must declare `metadata.name`:")
}
