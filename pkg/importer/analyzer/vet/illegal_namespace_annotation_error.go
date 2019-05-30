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
	status.AddExamples(IllegalNamespaceAnnotationErrorCode, IllegalNamespaceAnnotationError(role()))
}

var illegalNamespaceAnnotationError = status.NewErrorBuilder(IllegalNamespaceAnnotationErrorCode)

// IllegalNamespaceAnnotationError represents an illegal usage of the namespace selector annotation.
func IllegalNamespaceAnnotationError(resource id.Resource) status.Error {
	return illegalNamespaceAnnotationError.WithResources(resource).Errorf(
		"A %s MUST NOT use the annotation %s. "+
			"Remove metadata.annotations.%[2]s from:",
		node.Namespace, v1.NamespaceSelectorAnnotationKey)
}
