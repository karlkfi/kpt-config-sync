package nonhierarchical

import (
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/status"
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
