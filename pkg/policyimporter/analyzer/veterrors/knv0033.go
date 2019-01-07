package veterrors

import "github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1/repo"

// IllegalSystemResourcePlacementError reports that a nomos.dev object has been defined outside of system/
type IllegalSystemResourcePlacementError struct {
	ResourceID
}

// Error implements error
func (e IllegalSystemResourcePlacementError) Error() string {
	return format(e,
		"Resources of the below kind MUST NOT be declared outside %[1]s/:\n"+
			"%[2]s",
		repo.SystemDir, printResourceID(e))
}

// Code implements Error
func (e IllegalSystemResourcePlacementError) Code() string {
	return IllegalSystemResourcePlacementErrorCode
}
