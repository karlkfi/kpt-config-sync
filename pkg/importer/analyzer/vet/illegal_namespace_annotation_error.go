package vet

import (
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
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

var _ status.ResourceError = IllegalNamespaceAnnotationError{}

// Error implements error.
func (e IllegalNamespaceAnnotationError) Error() string {
	return status.Format(e,
		"A %s MUST NOT use the annotation %s. "+
			"Remove metadata.annotations.%[2]s from:",
		node.Namespace, v1.NamespaceSelectorAnnotationKey)
}

// Code implements Error
func (e IllegalNamespaceAnnotationError) Code() string {
	return IllegalNamespaceAnnotationErrorCode
}

// Resources implements ResourceError
func (e IllegalNamespaceAnnotationError) Resources() []id.Resource {
	return []id.Resource{e.Resource}
}

// ToCME implements ToCMEr.
func (e IllegalNamespaceAnnotationError) ToCME() v1.ConfigManagementError {
	return status.FromResourceError(e)
}
