package nonhierarchical

import (
	"github.com/golang/glog"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/util/discovery"
)

// IllegalNamespaceOnClusterScopedResourceErrorCode represents a cluster-scoped resource illegally
// declaring metadata.namespace.
const IllegalNamespaceOnClusterScopedResourceErrorCode = "1052"

var illegalNamespaceOnClusterScopedResourceErrorBuilder = status.NewErrorBuilder(IllegalNamespaceOnClusterScopedResourceErrorCode)

// IllegalNamespaceOnClusterScopedResourceError reports that a cluster-scoped resource MUST NOT declare metadata.namespace.
func IllegalNamespaceOnClusterScopedResourceError(resource id.Resource) status.Error {
	return illegalNamespaceOnClusterScopedResourceErrorBuilder.
		Sprint("cluster-scoped resources MUST NOT declare metadata.namespace").
		BuildWithResources(resource)
}

// MissingNamespaceOnNamespacedResourceErrorCode represents a namespace-scoped resource NOT declaring
// metadata.namespace.
const MissingNamespaceOnNamespacedResourceErrorCode = "1053"

var missingNamespaceOnNamespacedResourceErrorBuilder = status.NewErrorBuilder(MissingNamespaceOnNamespacedResourceErrorCode)

// NamespaceAndSelectorResourceError reports that a namespace-scoped resource illegally declares both metadata.namespace
// and has the namespace-selector annotation.
func NamespaceAndSelectorResourceError(resource id.Resource) status.Error {
	return missingNamespaceOnNamespacedResourceErrorBuilder.
		Sprintf("namespace-scoped resources MUST NOT declare both metadata.namespace and "+
			"metadata.annotations.%s", v1.NamespaceSelectorAnnotationKey).
		BuildWithResources(resource)
}

// MissingNamespaceOnNamespacedResourceError reports a namespace-scoped resource MUST declare metadata.namespace.
// when parsing in non-hierarchical mode.
func MissingNamespaceOnNamespacedResourceError(resource id.Resource) status.Error {
	return missingNamespaceOnNamespacedResourceErrorBuilder.
		Sprintf("namespace-scoped resources MUST either declare either metadata.namespace or "+
			"metadata.annotations.%s", v1.NamespaceSelectorAnnotationKey).
		BuildWithResources(resource)
}

// BadScopeErrCode is the error code indicating that a resource has been
// declared in a Namespace repository that shouldn't be there.
const BadScopeErrCode = "1058"

// BadScopeErrBuilder is an error build for errors related to the object scope errors
var BadScopeErrBuilder = status.NewErrorBuilder(BadScopeErrCode)

func shouldBeInRootErr(resource id.Resource) status.ResourceError {
	return BadScopeErrBuilder.
		Sprintf("Resources in namespace Repos must be Namespace-scoped type, but objects of type %v are Cluster-scoped. Move %s to the Root repo.",
			resource.GroupVersionKind(), resource.GetName()).
		BuildWithResources(resource)
}

// ScopeValidator returns errors for resources with illegal metadata.namespace
// declarations or cluster-scoped resources exist in the namespace scope.
//
// If the object is namespace-scoped and does not declare a NamespaceSelector,
// it is automatically assigned to the passed defaultNamespace.
func ScopeValidator(inNamespaceReconciler bool, defaultNamespace string, scoper discovery.Scoper, errOnUnknown bool) Validator {
	return PerObjectValidator(func(o ast.FileObject) status.Error {
		// Skip the validation when it is a Kptfile.
		// Kptfile is only for client side. A ResourceGroup CR will be generated from it
		// in subsequent program in pkg/parse.
		if o.GroupVersionKind().GroupKind() == kinds.KptFile().GroupKind() {
			return nil
		}
		scope, err := scoper.GetObjectScope(o)
		if err != nil {
			if errOnUnknown {
				return err
			}
			glog.V(6).Infof("ignored error due to --no-api-server-check: %s", err)
		}

		// namespace-scoped resources must declare either metadata.namespace or the
		// NamespaceSelector annotation when in nonhierarchical mode.
		hasNamespace := o.GetNamespace() != ""
		_, hasNamespaceSelector := o.GetAnnotations()[v1.NamespaceSelectorAnnotationKey]
		switch {
		case scope == discovery.NamespaceScope && hasNamespace:
			if hasNamespaceSelector {
				return NamespaceAndSelectorResourceError(o)
			}
			return nil
		case scope == discovery.NamespaceScope && !hasNamespace:
			if !hasNamespaceSelector {
				o.SetNamespace(defaultNamespace)
			}
			return nil
		case scope == discovery.ClusterScope && inNamespaceReconciler:
			return shouldBeInRootErr(&o)
		case scope == discovery.ClusterScope && hasNamespace:
			return IllegalNamespaceOnClusterScopedResourceError(&o)
		default:
			return nil
		}
	})
}
