package veterrors

import (
	"strings"

	"github.com/google/nomos/pkg/kinds"
)

// UnknownResourceVersionInSyncErrorCode is the error code for UnknownResourceVersionInSyncError
const UnknownResourceVersionInSyncErrorCode = "1039"

var unknownResourceVersionInSyncErrorExample = UnknownResourceVersionInSyncError{
	SyncID: &syncID{
		source:           "system/rq-sync.yaml",
		groupVersionKind: kinds.ResourceQuota().GroupKind().WithVersion("v2"),
	},
}

func init() {
	register(UnknownResourceVersionInSyncErrorCode, unknownResourceVersionInSyncErrorExample, "")
}

// UnknownResourceVersionInSyncError reports that a Sync contains a Group/Kind with an incorrect
// Version.
type UnknownResourceVersionInSyncError struct {
	SyncID
	Allowed []string
}

// Error implements error
func (e UnknownResourceVersionInSyncError) Error() string {
	return format(e,
		"Sync defines a Resource Kind with an incorrect Version. "+
			"Known Versions: [%[1]s]\n\n"+
			"%[2]s",
		strings.Join(e.Allowed, ", "), printSyncID(e))
}

// Code implements Error
func (e UnknownResourceVersionInSyncError) Code() string { return UnknownResourceVersionInSyncErrorCode }
