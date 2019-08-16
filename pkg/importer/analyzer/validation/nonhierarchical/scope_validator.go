package nonhierarchical

import (
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/util/discovery"
)

// IllegalNamespaceOnClusterScopedResourceErrorCode represents a cluster-scoped resource illegally
// declaring metadata.namespace.
const IllegalNamespaceOnClusterScopedResourceErrorCode = "1052"

var illegalNamespaceOnClusterScopedResourceErrorBuilder = status.NewErrorBuilder(IllegalNamespaceOnClusterScopedResourceErrorCode)

func illegalNamespaceOnClusterScopedResourceError(resource id.Resource) status.Error {
	return illegalNamespaceOnClusterScopedResourceErrorBuilder.WithResources(resource).
		New("cluster-scoped resources MUST NOT declare metadata.namespace")
}

// MissingNamespaceOnNamespacedResourceErrorCode represents a namespace-scoped resource NOT declaring
// metadata.namespace.
const MissingNamespaceOnNamespacedResourceErrorCode = "1053"

var missingNamespaceOnNamespacedResourceErrorBuilder = status.NewErrorBuilder(MissingNamespaceOnNamespacedResourceErrorCode)

func missingNamespaceOnNamespacedResourceError(resource id.Resource) status.Error {
	return missingNamespaceOnNamespacedResourceErrorBuilder.WithResources(resource).
		New("namespace-scoped resource MUST declare metadata.namespace")
}

// ScopeValidator returns errors for resources with illegal or missing metadata.namespace
// declarations.
func ScopeValidator(scoper discovery.Scoper) Validator {
	return perObjectValidator(func(o ast.FileObject) status.Error {
		switch scoper.GetScope(o.GroupVersionKind().GroupKind()) {
		case discovery.ClusterScope:
			if o.Namespace() != "" {
				return illegalNamespaceOnClusterScopedResourceError(&o)
			}
		case discovery.NamespaceScope:
			if o.Namespace() == "" {
				return missingNamespaceOnNamespacedResourceError(&o)
			}
		case discovery.UnknownScope:
			// Should be impossible to reach normally as an earlier validation should handle these cases.
			return status.InternalErrorf("type not registered on API server %q", o.GroupVersionKind().String())
		}
		return nil
	})
}
