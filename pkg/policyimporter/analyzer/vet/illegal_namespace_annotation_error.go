package vet

import (
	v1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast/node"
	"github.com/google/nomos/pkg/policyimporter/id"
	"github.com/google/nomos/pkg/status"
)

// IllegalNamespaceAnnotationErrorCode is the error code for IllegalNamespaceAnnotationError
const IllegalNamespaceAnnotationErrorCode = "1004"

func init() {
	register(IllegalNamespaceAnnotationErrorCode, nil, "")
}

// IllegalNamespaceAnnotationError represents an illegal usage of the namespace selector annotation.
type IllegalNamespaceAnnotationError struct {
	id.Resource
}

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
