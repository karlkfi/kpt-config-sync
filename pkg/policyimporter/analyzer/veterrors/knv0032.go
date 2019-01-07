package veterrors

// UnknownResourceInSyncError reports that a Resource defined in a Sync does not have a definition in the cluster.
type UnknownResourceInSyncError struct {
	SyncID
}

// Error implements error
func (e UnknownResourceInSyncError) Error() string {
	return format(e,
		"Sync defines a Resource Kind that does not exist on cluster. "+
			"Ensure the Group, Version, and Kind are spelled correctly.\n\n"+
			"%[1]s",
		printSyncID(e))
}

// Code implements Error
func (e UnknownResourceInSyncError) Code() string { return UnknownResourceInSyncErrorCode }
