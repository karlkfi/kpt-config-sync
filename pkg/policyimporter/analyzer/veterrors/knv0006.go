package veterrors

import "github.com/google/nomos/pkg/policyimporter/id"

// UnsyncableNamespaceObjectErrorCode is the error code for UnsyncableNamespaceObjectErrorCode
const UnsyncableNamespaceObjectErrorCode = "1006"

func init() {
	register(UnsyncableNamespaceObjectErrorCode, nil, "")
}

// UnsyncableNamespaceObjectError represents an illegal usage of a Resource which has not been defined for use in namespaces/.
type UnsyncableNamespaceObjectError struct {
	id.Resource
}

// Error implements error.
func (e UnsyncableNamespaceObjectError) Error() string {
	return format(e,
		"Unable to sync Resource. "+
			"Enable sync for this Resource's kind.\n\n"+
			"%[1]s",
		id.PrintResource(e))
}

// Code implements Error
func (e UnsyncableNamespaceObjectError) Code() string { return UnsyncableNamespaceObjectErrorCode }
