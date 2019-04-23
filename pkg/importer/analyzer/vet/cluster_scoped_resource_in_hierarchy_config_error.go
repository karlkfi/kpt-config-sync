package vet

import (
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/util/discovery"
)

// ClusterScopedResourceInHierarchyConfigErrorCode is the error code for ClusterScopedResourceInHierarchyConfigError
const ClusterScopedResourceInHierarchyConfigErrorCode = "1046"

func init() {
	status.Register(ClusterScopedResourceInHierarchyConfigErrorCode, ClusterScopedResourceInHierarchyConfigError{
		HierarchyConfig: fakeHierarchyConfig{
			Resource: hierarchyConfig(),
			gk:       kinds.Repo().GroupKind(),
		},
	})
}

// ClusterScopedResourceInHierarchyConfigError reports that a Resource defined in a HierarchyConfig
// has Cluster scope which means it's not feasible to interpret the resource in a hierarchical
// manner
type ClusterScopedResourceInHierarchyConfigError struct {
	id.HierarchyConfig
	Scope discovery.ObjectScope
}

var _ status.ResourceError = &ClusterScopedResourceInHierarchyConfigError{}

// Error implements error
func (e ClusterScopedResourceInHierarchyConfigError) Error() string {
	gk := e.GroupKind()
	return status.Format(e,
		"This HierarchyConfig references the APIResource %q which has %s scope. "+
			"Cluster scoped objects are not permitted in HierarchyConfig.",
		gk.String(),
		e.Scope)
}

// Code implements Error
func (e ClusterScopedResourceInHierarchyConfigError) Code() string {
	return ClusterScopedResourceInHierarchyConfigErrorCode
}

// Resources implements ResourceError
func (e ClusterScopedResourceInHierarchyConfigError) Resources() []id.Resource {
	return []id.Resource{e.HierarchyConfig}
}

// ToCME implements ToCMEr.
func (e ClusterScopedResourceInHierarchyConfigError) ToCME() v1.ConfigManagementError {
	return status.FromResourceError(e)
}
