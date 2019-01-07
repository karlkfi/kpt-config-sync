package veterrors

// UnsyncableNamespaceObjectError represents an illegal usage of a Resource which has not been defined for use in namespaces/.
type UnsyncableNamespaceObjectError struct {
	ResourceID
}

// Error implements error.
func (e UnsyncableNamespaceObjectError) Error() string {
	return format(e,
		"Unable to sync Resource. "+
			"Enable sync for this Resource's kind.\n\n"+
			"%[1]s",
		printResourceID(e))
}

// Code implements Error
func (e UnsyncableNamespaceObjectError) Code() string { return UnsyncableNamespaceObjectErrorCode }
