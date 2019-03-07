package vet

import (
	"strings"

	v1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/policyimporter/id"
	"github.com/google/nomos/pkg/status"
)

// IllegalHierarchyModeErrorCode is the error code for IllegalHierarchyModeError
const IllegalHierarchyModeErrorCode = "1042"

func init() {
	register(IllegalHierarchyModeErrorCode, nil, "")
}

// IllegalHierarchyModeError reports that a HierarchyConfig is defined with a disallowed hierarchyMode.
type IllegalHierarchyModeError struct {
	id.HierarchyConfig
	HierarchyMode v1.HierarchyModeType
	Allowed       map[v1.HierarchyModeType]bool
}

// Error implements error
func (e IllegalHierarchyModeError) Error() string {
	var allowedStr []string
	for a := range e.Allowed {
		allowedStr = append(allowedStr, string(a))
	}
	return status.Format(e,
		"HierarchyMode %[1]q is not a valid value for this Resource. Allowed values are [%[2]s].\n\n"+
			"%[3]s",
		e.HierarchyMode, strings.Join(allowedStr, ","), id.PrintHierarchyConfig(e))
}

// Code implements Error
func (e IllegalHierarchyModeError) Code() string { return IllegalHierarchyModeErrorCode }
