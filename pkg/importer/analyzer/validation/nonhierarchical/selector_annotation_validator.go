package nonhierarchical

import (
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/status"
)

// IllegalSelectorAnnotationErrorCode is the error code for IllegalNamespaceAnnotationError
const IllegalSelectorAnnotationErrorCode = "1004"

var illegalSelectorAnnotationError = status.NewErrorBuilder(IllegalSelectorAnnotationErrorCode)

// IllegalClusterSelectorAnnotationError reports that a Cluster or ClusterSelector declares the
// cluster-selector annotation.
func IllegalClusterSelectorAnnotationError(resource id.Resource, annotation string) status.Error {
	return illegalSelectorAnnotationError.
		Sprintf("%ss may not be cluster-selected, and so MUST NOT declare the annotation '%s'. "+
			"To fix, remove `metadata.annotations.%s` from:",
			resource.GroupVersionKind().Kind, annotation, annotation).
		BuildWithResources(resource)
}

// IllegalNamespaceSelectorAnnotationError reports that a cluster-scoped object declares the
// namespace-selector annotation.
func IllegalNamespaceSelectorAnnotationError(resource id.Resource) status.Error {
	return illegalSelectorAnnotationError.
		Sprintf("Cluster-scoped objects may not be namespace-selected, and so MUST NOT declare the annotation '%s'. "+
			"To fix, remove `metadata.annotations.%s` from:",
			v1.NamespaceSelectorAnnotationKey, v1.NamespaceSelectorAnnotationKey).
		BuildWithResources(resource)
}
