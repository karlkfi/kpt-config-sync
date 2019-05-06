package vet

import (
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/util/discovery"
)

// ClusterScopedResourceInHierarchyConfigErrorCode is the error code for ClusterScopedResourceInHierarchyConfigError
const ClusterScopedResourceInHierarchyConfigErrorCode = "1046"

func init() {
	status.AddExamples(ClusterScopedResourceInHierarchyConfigErrorCode, ClusterScopedResourceInHierarchyConfigError(
		fakeHierarchyConfig{
			Resource: hierarchyConfig(),
			gk:       kinds.ClusterSelector().GroupKind(),
		}, discovery.ClusterScope))
}

var clusterScopedResourceInHierarchyConfigError = status.NewErrorBuilder(ClusterScopedResourceInHierarchyConfigErrorCode)

// ClusterScopedResourceInHierarchyConfigError reports that a Resource defined in a HierarchyConfig
// has Cluster scope which means it's not feasible to interpret the resource in a hierarchical
// manner
func ClusterScopedResourceInHierarchyConfigError(config id.HierarchyConfig, scope discovery.ObjectScope) status.Error {
	gk := config.GroupKind()
	return clusterScopedResourceInHierarchyConfigError.WithResources(config).Errorf(
		"This HierarchyConfig references the APIResource %q which has %s scope. "+
			"Cluster scoped objects are not permitted in HierarchyConfig.",
		gk.String(), scope)
}
