package nonhierarchical

import (
	"github.com/golang/glog"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/util/discovery"
)

// NewClusterSelectorAnnotationValidator ensures objects do not declare invalid cluster-selector annotations.
func NewClusterSelectorAnnotationValidator() Validator {
	return PerObjectValidator(func(o ast.FileObject) status.Error {
		return validateClusterSelectorAnnotation(o)
	})
}

// NewNamespaceSelectorAnnotationValidator ensures objects do not declare invalid namespace-selector annotations.
func NewNamespaceSelectorAnnotationValidator(scoper discovery.Scoper, errorOnUnknown bool) Validator {
	return PerObjectValidator(func(o ast.FileObject) status.Error {
		return validateNamespaceSelectorAnnotation(scoper, o, errorOnUnknown)
	})
}

func validateClusterSelectorAnnotation(o ast.FileObject) status.Error {
	if !forbidsSelectors(o) {
		return nil
	}

	if _, hasAnnotation := o.GetAnnotations()[v1.LegacyClusterSelectorAnnotationKey]; hasAnnotation {
		// This is a Cluster, ClusterSelector, or NamespaceSelector, and it defines the legacy cluster-selector annotation.
		return IllegalClusterSelectorAnnotationError(o, v1.LegacyClusterSelectorAnnotationKey)
	}
	if _, hasAnnotation := o.GetAnnotations()[v1alpha1.ClusterNameSelectorAnnotationKey]; hasAnnotation {
		// This is a Cluster, ClusterSelector, or NamespaceSelector, and it defines the inline cluster-selector annotation.
		return IllegalClusterSelectorAnnotationError(o, v1alpha1.ClusterNameSelectorAnnotationKey)
	}
	return nil
}

func validateNamespaceSelectorAnnotation(scoper discovery.Scoper, o ast.FileObject, errOnUnknown bool) status.Error {
	// Namespace-scoped objects may declare the namespace-selector annotation.

	// Skip the validation when it is a Kptfile.
	// Kptfile is only for client side. A ResourceGroup CR will be generated from it
	// in subsequent program in pkg/parse.
	if o.GroupVersionKind().GroupKind() == kinds.KptFile().GroupKind() {
		return nil
	}
	if !forbidsSelectors(o) {
		isNamespaced, err := scoper.GetObjectScope(o)
		if err != nil {
			if errOnUnknown {
				return err
			}
			glog.V(6).Infof("ignored error due to --no-api-server-check: %s", err)
			return nil
		}
		if isNamespaced == discovery.NamespaceScope {
			return nil
		}
	}

	if _, hasAnnotation := o.GetAnnotations()[v1.NamespaceSelectorAnnotationKey]; hasAnnotation {
		// This is cluster-scoped, and it defines the namespace-selector annotation.
		return IllegalNamespaceSelectorAnnotationError(o)
	}
	return nil
}

func forbidsSelectors(o ast.FileObject) bool {
	// Cluster, ClusterSelector, and NamespaceSelector aren't necessarily defined on the APIServer,
	// and we should verify they don't have the NamespaceSelector annotation.
	gvk := o.GroupVersionKind()
	return gvk == kinds.Cluster() ||
		gvk == kinds.ClusterSelector() ||
		gvk == kinds.NamespaceSelector() ||
		gvk.GroupKind() == kinds.CustomResourceDefinition()
}

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
