package vet

import (
	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/ast/node"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/status"
)

// IllegalNamespaceAnnotationErrorCode is the error code for IllegalNamespaceAnnotationError
const IllegalNamespaceAnnotationErrorCode = "1004"

func init() {
	status.Register(IllegalNamespaceAnnotationErrorCode, IllegalNamespaceAnnotationError{Resource: role()})
}

// IllegalNamespaceAnnotationError represents an illegal usage of the namespace selector annotation.
type IllegalNamespaceAnnotationError struct {
	id.Resource
}

var _ status.ResourceError = &IllegalNamespaceAnnotationError{}

// Error implements error.
func (e IllegalNamespaceAnnotationError) Error() string {
	return status.Format(e,
		"A %[3]s MUST NOT use the annotation %[2]s. "+
			"Remove metadata.annotations.%[2]s from:\n\n"+
			"%[1]s",
		id.PrintResource(e.Resource), v1.NamespaceSelectorAnnotationKey, node.Namespace)
}

// Code implements Error
func (e IllegalNamespaceAnnotationError) Code() string {
	return IllegalNamespaceAnnotationErrorCode
}

// Resources implements ResourceError
func (e IllegalNamespaceAnnotationError) Resources() []id.Resource {
	return []id.Resource{e.Resource}
}
