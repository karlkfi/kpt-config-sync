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
	status.AddExamples(UnknownResourceInHierarchyConfigErrorCode, UnknownResourceInHierarchyConfigError(
		fakeHierarchyConfig{
			Resource: hierarchyConfig(),
			gk:       kinds.Repo().GroupKind(),
		},
	))
}

type fakeHierarchyConfig struct {
	id.Resource
	gk schema.GroupKind
}

// GroupKind implements id.HierarchyConfig.
func (hc fakeHierarchyConfig) GroupKind() schema.GroupKind {
	return hc.gk
}

var unknownResourceInHierarchyConfigError = status.NewErrorBuilder(UnknownResourceInHierarchyConfigErrorCode)

// UnknownResourceInHierarchyConfigError reports that a Resource defined in a HierarchyConfig does not have a definition in
// the cluster.
func UnknownResourceInHierarchyConfigError(config id.HierarchyConfig) status.Error {
	gk := config.GroupKind()
	return unknownResourceInHierarchyConfigError.WithResources(config).Errorf(
		"This HierarchyConfig defines the APIResource %q which does not exist on cluster. "+
			"Ensure the Group and Kind are spelled correctly and any required CRD exists on the cluster.",
		gk.String())
}
