package vet

import (
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
)

// UnsupportedResourceInHierarchyConfigErrorCode is the error code for UnsupportedResourceInHierarchyConfigError
const UnsupportedResourceInHierarchyConfigErrorCode = "1041"

func init() {
	status.Register(UnsupportedResourceInHierarchyConfigErrorCode, UnsupportedResourceInHierarchyConfigError{
		HierarchyConfig: fakeHierarchyConfig{
			Resource: hierarhcyConfig(),
			gk:       kinds.Repo().GroupKind(),
		},
	})
}

// UnsupportedResourceInHierarchyConfigError reports that policy management is unsupported for a Resource defined in a HierarchyConfig.
type UnsupportedResourceInHierarchyConfigError struct {
	id.HierarchyConfig
}

var _ id.ResourceError = &UnsupportedResourceInHierarchyConfigError{}

// Error implements error
func (e UnsupportedResourceInHierarchyConfigError) Error() string {
	return status.Format(e,
		"This Resource Kind MUST NOT be declared in a HierarchyConfig:\n\n"+
			"%[1]s",
		id.PrintHierarchyConfig(e))
}

// Code implements Error
func (e UnsupportedResourceInHierarchyConfigError) Code() string {
	return UnsupportedResourceInHierarchyConfigErrorCode
}

// Resources implements ResourceError
func (e UnsupportedResourceInHierarchyConfigError) Resources() []id.Resource {
	return []id.Resource{e.HierarchyConfig}
}
