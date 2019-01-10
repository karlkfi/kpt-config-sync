package veterrors

import "github.com/google/nomos/pkg/policyimporter/id"

// UnsupportedResourceInSyncErrorCode is the error code for UnsupportedResourceInSyncError
const UnsupportedResourceInSyncErrorCode = "1034"

func init() {
	register(UnsupportedResourceInSyncErrorCode, nil, "")
}

// UnsupportedResourceInSyncError reports that policy management is unsupported for a Resource defined in a Sync.
type UnsupportedResourceInSyncError struct {
	id.Sync
}

// Error implements error
func (e UnsupportedResourceInSyncError) Error() string {
	return format(e,
		"This Resource Kind MUST NOT be declared in a Sync:\n\n"+
			"%[1]s",
		id.PrintSync(e))
}

// Code implements Error
func (e UnsupportedResourceInSyncError) Code() string { return UnsupportedResourceInSyncErrorCode }
