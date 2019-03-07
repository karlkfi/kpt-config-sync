package vet

import (
	"github.com/google/nomos/pkg/policyimporter/id"
	"github.com/google/nomos/pkg/status"
)

// UnsupportedObjectErrorCode is the error code for UnsupportedObjectError
const UnsupportedObjectErrorCode = "1043"

func init() {
	register(UnsupportedObjectErrorCode, nil, "")
}

// UnsupportedObjectError reports than an unsupported object is in the namespaces/ sub-directories or clusters/ directory.
type UnsupportedObjectError struct {
	id.Resource
}

// Error implements error.
func (e UnsupportedObjectError) Error() string {
	return status.Format(e,
		"Syncing is not supported for this resource. To fix, remove this resource from the repo.\n\n"+
			"%[1]s",
		id.PrintResource(e))
}

// Code implements Error
func (e UnsupportedObjectError) Code() string { return UnsupportedObjectErrorCode }
