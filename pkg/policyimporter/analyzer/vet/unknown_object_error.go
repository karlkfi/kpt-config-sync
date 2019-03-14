package vet

import (
	"github.com/google/nomos/pkg/policyimporter/id"
	"github.com/google/nomos/pkg/status"
)

// UnknownObjectErrorCode is the error code for UnknownObjectError
const UnknownObjectErrorCode = "1021" // Impossible to create consistent example.

func init() {
	register(UnknownObjectErrorCode)
}

// UnknownObjectError reports that an object declared in the repo does not have a definition in the cluster.
type UnknownObjectError struct {
	id.Resource
}

var _ id.ResourceError = &UnknownObjectError{}

// Error implements error
func (e UnknownObjectError) Error() string {
	return status.Format(e,
		"No CustomResourceDefinition is defined for the resource in the cluster. "+
			"\nResource types that are not native Kubernetes objects must have a CustomResourceDefinition.\n\n%s",
		id.PrintResource(e))
}

// Code implements Error
func (e UnknownObjectError) Code() string { return UnknownObjectErrorCode }

// Resources implements ResourceError
func (e UnknownObjectError) Resources() []id.Resource {
	return []id.Resource{e.Resource}
}
