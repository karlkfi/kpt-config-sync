package vet

import (
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/status"
)

// UnknownObjectErrorCode is the error code for UnknownObjectError
const UnknownObjectErrorCode = "1021" // Impossible to create consistent example.

func init() {
	status.AddExamples(UnknownObjectErrorCode, UnknownObjectError(
		role(),
	))
}

var unknownObjectError = status.NewErrorBuilder(UnknownObjectErrorCode)

// UnknownObjectError reports that an object declared in the repo does not have a definition in the cluster.
func UnknownObjectError(resource id.Resource) status.Error {
	return unknownObjectError.WithResources(resource).New(
		"No CustomResourceDefinition is defined for the resource in the cluster. " +
			"\nResource types that are not native Kubernetes objects must have a CustomResourceDefinition.")
}
