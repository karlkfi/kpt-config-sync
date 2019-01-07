package veterrors

import (
	"strings"

	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
)

// IllegalHierarchyModeError reports that a Sync is defined with a disallowed hierarchyMode.
type IllegalHierarchyModeError struct {
	SyncID
	HierarchyMode v1alpha1.HierarchyModeType
	Allowed       map[v1alpha1.HierarchyModeType]bool
}

// Error implements error
func (e IllegalHierarchyModeError) Error() string {
	var allowedStr []string
	for a := range e.Allowed {
		allowedStr = append(allowedStr, string(a))
	}
	return format(e,
		"HierarchyMode %[1]q is not a valid value for this Resource. Allowed values are [%[2]s].\n\n"+
			"%[3]s",
		e.HierarchyMode, strings.Join(allowedStr, ","), printSyncID(e))
}

// Code implements Error
func (e IllegalHierarchyModeError) Code() string { return IllegalHierarchyModeErrorCode }
