package veterrors

// UnknownObjectError reports that an object declared in the repo does not have a definition in the cluster.
type UnknownObjectError struct {
	ResourceID
}

// Error implements error
func (e UnknownObjectError) Error() string {
	return format(e,
		"Transient Error: Resource is declared, but has no definition on the cluster."+
			"\nResource must be a native K8S Resources or have an associated CustomResourceDefinition:\n\n%s",
		printResourceID(e))
}

// Code implements Error
func (e UnknownObjectError) Code() string { return UnknownObjectErrorCode }
