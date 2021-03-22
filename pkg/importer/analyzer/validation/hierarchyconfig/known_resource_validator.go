package hierarchyconfig

import (
	"github.com/google/nomos/pkg/status"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ClusterScopedResourceInHierarchyConfigErrorCode is the error code for ClusterScopedResourceInHierarchyConfigError
const ClusterScopedResourceInHierarchyConfigErrorCode = "1046"

var clusterScopedResourceInHierarchyConfigError = status.NewErrorBuilder(ClusterScopedResourceInHierarchyConfigErrorCode)

// ClusterScopedResourceInHierarchyConfigError reports that a Resource defined in a HierarchyConfig
// has Cluster scope which means it's not feasible to interpret the resource in a hierarchical
// manner
func ClusterScopedResourceInHierarchyConfigError(config client.Object, gk schema.GroupKind) status.Error {
	return clusterScopedResourceInHierarchyConfigError.
		Sprintf("This HierarchyConfig references the APIResource %q which has Cluster scope. "+
			"Cluster scoped objects are not permitted in HierarchyConfig.",
			gk.String()).
		BuildWithResources(config)
}
