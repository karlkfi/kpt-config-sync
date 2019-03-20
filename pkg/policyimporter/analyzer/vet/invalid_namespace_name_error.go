package vet

import (
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast/node"
	"github.com/google/nomos/pkg/policyimporter/id"
	"github.com/google/nomos/pkg/status"
)

// InvalidNamespaceNameErrorCode is the error code for InvalidNamespaceNameError
const InvalidNamespaceNameErrorCode = "1020"

func init() {
	status.Register(InvalidNamespaceNameErrorCode, InvalidNamespaceNameError{})
}

// InvalidNamespaceNameError reports that a Namespace has an invalid name.
type InvalidNamespaceNameError struct {
	id.Resource
	Expected string
}

var _ id.ResourceError = &InvalidNamespaceNameError{}

// Error implements error
func (e InvalidNamespaceNameError) Error() string {
	return status.Format(e,
		"A %[1]s MUST declare metadata.name that matches the name of its directory.\n\n"+
			"%[2]s\n\n"+
			"expected metadata.name: %[3]s\n",
		node.Namespace, id.PrintResource(e), e.Expected)
}

// Code implements Error
func (e InvalidNamespaceNameError) Code() string { return InvalidNamespaceNameErrorCode }

// Resources implements ResourceError
func (e InvalidNamespaceNameError) Resources() []id.Resource {
	return []id.Resource{e.Resource}
}
