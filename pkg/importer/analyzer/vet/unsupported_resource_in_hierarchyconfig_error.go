package vet

import (
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
)

// UnsupportedResourceInHierarchyConfigErrorCode is the error code for UnsupportedResourceInHierarchyConfigError
const UnsupportedResourceInHierarchyConfigErrorCode = "1041"

func init() {
	status.Register(UnsupportedResourceInHierarchyConfigErrorCode, UnsupportedResourceInHierarchyConfigError{
		HierarchyConfig: fakeHierarchyConfig{
			Resource: hierarchyConfig(),
			gk:       kinds.Repo().GroupKind(),
		},
	})
}

// UnsupportedResourceInHierarchyConfigError reports that policy management is unsupported for a Resource defined in a HierarchyConfig.
type UnsupportedResourceInHierarchyConfigError struct {
	id.HierarchyConfig
}

var _ status.ResourceError = &UnsupportedResourceInHierarchyConfigError{}

// Error implements error
func (e UnsupportedResourceInHierarchyConfigError) Error() string {
	gk := e.GroupKind()
	return status.Format(e,
		"The %q APIResource MUST NOT be declared in a HierarchyConfig:",
		gk.String())
}

// Code implements Error
func (e UnsupportedResourceInHierarchyConfigError) Code() string {
	return UnsupportedResourceInHierarchyConfigErrorCode
}

// Resources implements ResourceError
func (e UnsupportedResourceInHierarchyConfigError) Resources() []id.Resource {
	return []id.Resource{e.HierarchyConfig}
}

// ToCME implements ToCMEr.
func (e UnsupportedResourceInHierarchyConfigError) ToCME() v1.ConfigManagementError {
	return status.FromResourceError(e)
}
