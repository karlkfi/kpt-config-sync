package veterrors

// MissingObjectNameError reports that an object has no name.
type MissingObjectNameError struct {
	ResourceID
}

// Error implements error
func (e MissingObjectNameError) Error() string {
	return format(e,
		"Resources must declare metadata.name:\n\n"+
			"%[1]s",
		printResourceID(e))
}

// Code implements Error
func (e MissingObjectNameError) Code() string { return MissingObjectNameErrorCode }
