package vet

import "github.com/google/nomos/pkg/policyimporter/id"

// UnknownObjectErrorCode is the error code for UnknownObjectError
const UnknownObjectErrorCode = "1021" // Impossible to create consistent example.

func init() {
	register(UnknownObjectErrorCode, nil, "")
}

// UnknownObjectError reports that an object declared in the repo does not have a definition in the cluster.
type UnknownObjectError struct {
	id.Resource
}

// Error implements error
func (e UnknownObjectError) Error() string {
	return format(e,
		"No CustomResourceDefinition is defined for the resource in the cluster. "+
			"\nResource types that are not native Kubernetes objects must have a CustomResourceDefinition.\n\n%s",
		id.PrintResource(e))
}

// Code implements Error
func (e UnknownObjectError) Code() string { return UnknownObjectErrorCode }
