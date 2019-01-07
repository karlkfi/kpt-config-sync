package veterrors

import "strings"

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
