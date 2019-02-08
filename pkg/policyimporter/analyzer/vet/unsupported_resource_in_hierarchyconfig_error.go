package vet

import "github.com/google/nomos/pkg/policyimporter/id"

// UnsupportedResourceInHierarchyConfigErrorCode is the error code for UnsupportedResourceInHierarchyConfigError
const UnsupportedResourceInHierarchyConfigErrorCode = "1041"

func init() {
	register(UnsupportedResourceInHierarchyConfigErrorCode, nil, "")
}

// UnsupportedResourceInHierarchyConfigError reports that policy management is unsupported for a Resource defined in a HierarchyConfig.
type UnsupportedResourceInHierarchyConfigError struct {
	id.HierarchyConfig
}

// Error implements error
func (e UnsupportedResourceInHierarchyConfigError) Error() string {
	return format(e,
		"This Resource Kind MUST NOT be declared in a HierarchyConfig:\n\n"+
			"%[1]s",
		id.PrintHierarchyConfig(e))
}

// Code implements Error
func (e UnsupportedResourceInHierarchyConfigError) Code() string {
	return UnsupportedResourceInHierarchyConfigErrorCode
}
