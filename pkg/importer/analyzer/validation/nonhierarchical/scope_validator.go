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

// MissingNamespaceOnNamespacedResourceError reports a namespace-scoped resource MUST declare metadata.namespace.
// when parsing in non-hierarchical mode.
func MissingNamespaceOnNamespacedResourceError(resource id.Resource) status.Error {
	return missingNamespaceOnNamespacedResourceErrorBuilder.
		Sprint("namespace-scoped resource MUST declare metadata.namespace").
		BuildWithResources(resource)
}

// ScopeValidator returns errors for resources with illegal or missing metadata.namespace
// declarations.
func ScopeValidator(scoper discovery.Scoper) Validator {
	return PerObjectValidator(func(o ast.FileObject) status.Error {
		switch scoper.GetScope(o.GroupVersionKind().GroupKind()) {
		case discovery.ClusterScope:
			if o.GetNamespace() != "" {
				return IllegalNamespaceOnClusterScopedResourceError(&o)
			}
		case discovery.NamespaceScope:
			if o.GetNamespace() == "" {
				return MissingNamespaceOnNamespacedResourceError(&o)
			}
		case discovery.UnknownScope:
			// Should be impossible to reach normally as an earlier validation should handle these cases.
			return status.InternalErrorf("type not registered on API server %q", o.GroupVersionKind().String())
		}
		return nil
	})
}
