package veterrors

import "path"

// ReservedDirectoryNameErrorCode is the error code for ReservedDirectoryNameError
const ReservedDirectoryNameErrorCode = "1001"

var reservedDirectoryNameErrorExample = ReservedDirectoryNameError{Dir: "namespaces/reserved"}

var reservedDirectoryNameExplanation = `
GKE Policy Management defines several
[Reserved Namespaces](../management_flow.md#namespaces), and users may
[specify their own Reserved Namespaces](../system_config.md#reserved-namespaces).
Namespace and Abstract Namespace directories MUST NOT use these reserved names.
To fix:

1.  rename the directory,
1.  remove the directory, or
1.  remove the reserved namespace declaration.
`

func init() {
	register(ReservedDirectoryNameErrorCode, reservedDirectoryNameErrorExample, reservedDirectoryNameExplanation)
}

// ReservedDirectoryNameError represents an illegal usage of a reserved name.
type ReservedDirectoryNameError struct {
	Dir string
}

// Error implements error.
func (e ReservedDirectoryNameError) Error() string {
	return format(e,
		"Directories MUST NOT have reserved namespace names. Rename or remove directory:\n\n"+
			"path: %[1]s\n"+
			"name: %[2]s",
		e.Dir, path.Base(e.Dir))
}

// Code implements Error
func (e ReservedDirectoryNameError) Code() string { return ReservedDirectoryNameErrorCode }
