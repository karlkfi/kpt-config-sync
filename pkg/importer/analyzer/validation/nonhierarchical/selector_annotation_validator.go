package nonhierarchical

import (
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/util/discovery"
)

// NewSelectorAnnotationValidator ensures objects do not declare invalid namespace-selector
// and cluster-selector annotations.
func NewSelectorAnnotationValidator(scoper discovery.Scoper) Validator {
	return PerObjectValidator(func(o ast.FileObject) status.Error {
		csErr := validateClusterSelectorAnnotation(o)
		if csErr != nil {
			return csErr
		}
		return validateNamespaceSelectorAnnotation(scoper, o)
	})
}

func validateClusterSelectorAnnotation(o ast.FileObject) status.Error {
	gvk := o.GroupVersionKind()
	// All types except Cluster, ClusterSelector, and NamespaceSelector may declare the cluster-selector annotation.
	if gvk != kinds.Cluster() && gvk != kinds.ClusterSelector() && gvk != kinds.NamespaceSelector() {
		return nil
	}
	if _, hasAnnotation := o.GetAnnotations()[v1.ClusterSelectorAnnotationKey]; hasAnnotation {
		// This is a Cluster, ClusterSelector, or NamespaceSelector, and it defines the cluster-selector annotation.
		return IllegalClusterSelectorAnnotationError(o)
	}
	return nil
}

func validateNamespaceSelectorAnnotation(scoper discovery.Scoper, o ast.FileObject) status.Error {
	// Namespace-scoped objects may declare the namespace-selector annotation.
	if scoper.GetScope(o.GroupVersionKind().GroupKind()) == discovery.NamespaceScope {
		return nil
	}
	if _, hasAnnotation := o.GetAnnotations()[v1.NamespaceSelectorAnnotationKey]; hasAnnotation {
		// This is cluster-scoped, and it defines the namespace-selector annotation.
		return IllegalNamespaceSelectorAnnotationError(o)
	}
	return nil
}

// IllegalSelectorAnnotationErrorCode is the error code for IllegalNamespaceAnnotationError
const IllegalSelectorAnnotationErrorCode = "1004"

var illegalSelectorAnnotationError = status.NewErrorBuilder(IllegalSelectorAnnotationErrorCode)

// IllegalClusterSelectorAnnotationError reports that a Cluster or ClusterSelector declares the
// cluster-selector annotation.
func IllegalClusterSelectorAnnotationError(resource id.Resource) status.Error {
	return illegalSelectorAnnotationError.
		Sprintf("%ss may not be cluster-selected, and so MUST NOT declare the annotation %s. "+
			"To fix, remove metadata.annotations.%s from:",
			resource.GroupVersionKind().Kind, v1.ClusterSelectorAnnotationKey).
		BuildWithResources(resource)
}

// IllegalNamespaceSelectorAnnotationError reports that a cluster-scoped object declares the
// namespace-selector annotation.
func IllegalNamespaceSelectorAnnotationError(resource id.Resource) status.Error {
	return illegalSelectorAnnotationError.
		Sprintf("Cluster-scoped objects may not be namespace-selected, and so MUST NOT declare the annotation %s. "+
			"To fix, remove metadata.annotations.%s from:",
			v1.NamespaceSelectorAnnotationKey).
		BuildWithResources(resource)
}
