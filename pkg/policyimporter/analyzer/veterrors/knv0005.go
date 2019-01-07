package veterrors

// UnsyncableClusterObjectError represents an illegal usage of a cluster object kind which has not be explicitly declared.
type UnsyncableClusterObjectError struct {
	ResourceID
}

// Error implements error.
func (e UnsyncableClusterObjectError) Error() string {
	return format(e,
		"Unable to sync Resource. Enable sync for this Resource's kind.\n\n"+
			"%[1]s",
		printResourceID(e))
}

// Code implements Error
func (e UnsyncableClusterObjectError) Code() string { return UnsyncableClusterObjectErrorCode }
