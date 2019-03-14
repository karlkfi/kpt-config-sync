package vet

import (
	"github.com/google/nomos/pkg/policyimporter/id"
	"github.com/google/nomos/pkg/status"
)

// UnsyncableClusterObjectErrorCode is the error code for UnsyncableClusterObjectErrorCode
const UnsyncableClusterObjectErrorCode = "1005"

func init() {
	register(UnsyncableClusterObjectErrorCode)
}

// UnsyncableClusterObjectError represents an illegal usage of a cluster object kind which has not be explicitly declared.
type UnsyncableClusterObjectError struct {
	id.Resource
}

var _ id.ResourceError = &UnsyncableClusterObjectError{}

// Error implements error.
func (e UnsyncableClusterObjectError) Error() string {
	return status.Format(e,
		"Unable to sync Resource. Enable sync for this Resource's kind.\n\n"+
			"%[1]s",
		id.PrintResource(e))
}

// Code implements Error
func (e UnsyncableClusterObjectError) Code() string { return UnsyncableClusterObjectErrorCode }

// Resources implements ResourceError
func (e UnsyncableClusterObjectError) Resources() []id.Resource {
	return []id.Resource{e.Resource}
}
