package vet

import (
	"github.com/google/nomos/pkg/policyimporter/id"
	"github.com/google/nomos/pkg/status"
)

// UnknownResourceInHierarchyConfigErrorCode is the error code for UnknownResourceInHierarchyConfigError
const UnknownResourceInHierarchyConfigErrorCode = "1040"

func init() {
	status.Register(UnknownResourceInHierarchyConfigErrorCode, UnknownResourceInHierarchyConfigError{})
}

// UnknownResourceInHierarchyConfigError reports that a Resource defined in a HierarchyConfig does not have a definition in
// the cluster.
type UnknownResourceInHierarchyConfigError struct {
	id.HierarchyConfig
}

var _ id.ResourceError = &UnknownResourceInHierarchyConfigError{}

// Error implements error
func (e UnknownResourceInHierarchyConfigError) Error() string {
	return status.Format(e,
		"HierarchyConfig defines a Resource Kind that does not exist on cluster. "+
			"Ensure the Group and Kind are spelled correctly.\n\n"+
			"%[1]s",
		id.PrintHierarchyConfig(e))
}

// Code implements Error
func (e UnknownResourceInHierarchyConfigError) Code() string {
	return UnknownResourceInHierarchyConfigErrorCode
}

// Resources implements ResourceError
func (e UnknownResourceInHierarchyConfigError) Resources() []id.Resource {
	return []id.Resource{e.HierarchyConfig}
}
