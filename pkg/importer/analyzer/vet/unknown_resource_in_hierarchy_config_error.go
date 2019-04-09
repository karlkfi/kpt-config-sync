package vet

import (
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// UnknownResourceInHierarchyConfigErrorCode is the error code for UnknownResourceInHierarchyConfigError
const UnknownResourceInHierarchyConfigErrorCode = "1040"

func init() {
	status.Register(UnknownResourceInHierarchyConfigErrorCode, UnknownResourceInHierarchyConfigError{
		HierarchyConfig: fakeHierarchyConfig{
			Resource: hierarchyConfig(),
			gk:       kinds.Repo().GroupKind(),
		},
	})
}

type fakeHierarchyConfig struct {
	id.Resource
	gk schema.GroupKind
}

// GroupKind implements id.HierarchyConfig.
func (hc fakeHierarchyConfig) GroupKind() schema.GroupKind {
	return hc.gk
}

// UnknownResourceInHierarchyConfigError reports that a Resource defined in a HierarchyConfig does not have a definition in
// the cluster.
type UnknownResourceInHierarchyConfigError struct {
	id.HierarchyConfig
}

var _ status.ResourceError = &UnknownResourceInHierarchyConfigError{}

// Error implements error
func (e UnknownResourceInHierarchyConfigError) Error() string {
	gk := e.GroupKind()
	return status.Format(e,
		"This HierarchyConfig defines the APIResource %q which does not exist on cluster. "+
			"Ensure the Group and Kind are spelled correctly and any required CRD exists on the cluster.",
		gk.String())
}

// Code implements Error
func (e UnknownResourceInHierarchyConfigError) Code() string {
	return UnknownResourceInHierarchyConfigErrorCode
}

// Resources implements ResourceError
func (e UnknownResourceInHierarchyConfigError) Resources() []id.Resource {
	return []id.Resource{e.HierarchyConfig}
}
