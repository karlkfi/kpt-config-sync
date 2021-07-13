package nonhierarchical

import (
	"github.com/google/nomos/pkg/metadata"
	"github.com/google/nomos/pkg/status"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// IllegalSelectorAnnotationErrorCode is the error code for IllegalNamespaceAnnotationError
const IllegalSelectorAnnotationErrorCode = "1004"

var illegalSelectorAnnotationError = status.NewErrorBuilder(IllegalSelectorAnnotationErrorCode)

// IllegalClusterSelectorAnnotationError reports that a Cluster or ClusterSelector declares the
// cluster-selector annotation.
func IllegalClusterSelectorAnnotationError(resource client.Object, annotation string) status.Error {
	return illegalSelectorAnnotationError.
		Sprintf("%ss may not be cluster-selected, and so MUST NOT declare the annotation '%s'. "+
			"To fix, remove `metadata.annotations.%s` from:",
			resource.GetObjectKind().GroupVersionKind().Kind, annotation, annotation).
		BuildWithResources(resource)
}

// IllegalNamespaceSelectorAnnotationError reports that a cluster-scoped object declares the
// namespace-selector annotation.
func IllegalNamespaceSelectorAnnotationError(resource client.Object) status.Error {
	return illegalSelectorAnnotationError.
		Sprintf("Cluster-scoped objects may not be namespace-selected, and so MUST NOT declare the annotation '%s'. "+
			"To fix, remove `metadata.annotations.%s` from:",
			metadata.NamespaceSelectorAnnotationKey, metadata.NamespaceSelectorAnnotationKey).
		BuildWithResources(resource)
}
