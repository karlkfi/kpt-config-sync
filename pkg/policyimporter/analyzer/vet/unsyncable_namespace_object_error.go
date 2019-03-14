package vet

import (
	"github.com/google/nomos/pkg/policyimporter/id"
	"github.com/google/nomos/pkg/status"
)

// UnsyncableNamespaceObjectErrorCode is the error code for UnsyncableNamespaceObjectErrorCode
const UnsyncableNamespaceObjectErrorCode = "1006"

func init() {
	register(UnsyncableNamespaceObjectErrorCode)
}

// UnsyncableNamespaceObjectError represents an illegal usage of a Resource which has not been defined for use in namespaces/.
type UnsyncableNamespaceObjectError struct {
	id.Resource
}

var _ id.ResourceError = &UnsyncableNamespaceObjectError{}

// Error implements error.
func (e UnsyncableNamespaceObjectError) Error() string {
	return status.Format(e,
		"Unable to sync Resource. "+
			"Enable sync for this Resource's kind.\n\n"+
			"%[1]s",
		id.PrintResource(e))
}

// Code implements Error
func (e UnsyncableNamespaceObjectError) Code() string { return UnsyncableNamespaceObjectErrorCode }

// Resources implements ResourceError
func (e UnsyncableNamespaceObjectError) Resources() []id.Resource {
	return []id.Resource{e.Resource}
}
