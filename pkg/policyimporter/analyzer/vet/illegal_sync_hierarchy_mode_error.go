package vet

import (
	"strings"

	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/policyimporter/id"
)

// IllegalSyncHierarchyModeErrorCode is the error code for IllegalSyncHierarchyModeError
const IllegalSyncHierarchyModeErrorCode = "1035"

func init() {
	register(IllegalSyncHierarchyModeErrorCode, nil, "")
}

// IllegalSyncHierarchyModeError reports that a Sync is defined with a disallowed hierarchyMode.
type IllegalSyncHierarchyModeError struct {
	id.Sync
	HierarchyMode v1alpha1.HierarchyModeType
	Allowed       map[v1alpha1.HierarchyModeType]bool
}

// Error implements error
func (e IllegalSyncHierarchyModeError) Error() string {
	var allowedStr []string
	for a := range e.Allowed {
		allowedStr = append(allowedStr, string(a))
	}
	return format(e,
		"HierarchyMode %[1]q is not a valid value for this Resource. Allowed values are [%[2]s].\n\n"+
			"%[3]s",
		e.HierarchyMode, strings.Join(allowedStr, ","), id.PrintSync(e))
}

// Code implements Error
func (e IllegalSyncHierarchyModeError) Code() string { return IllegalSyncHierarchyModeErrorCode }
