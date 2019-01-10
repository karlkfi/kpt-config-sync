package veterrors

import (
	"github.com/google/nomos/pkg/policyimporter/filesystem/nomospath"
)

// ReservedDirectoryNameErrorCode is the error code for ReservedDirectoryNameError
const ReservedDirectoryNameErrorCode = "1001"

var reservedDirectoryNameErrorExamples = []Error{ReservedDirectoryNameError{Dir: nomospath.NewFakeRelative("namespaces/reserved")}}

var reservedDirectoryNameExplanation = `
GKE Policy Management defines several
[Reserved Namespaces](../management_flow.md#namespaces), and users may
[specify their own Reserved Namespaces](../system_config.md#reserved-namespaces).
Namespace and Abstract Namespace directories MUST NOT use these reserved names.
To fix:

1.  rename the directory,
1.  remove the directory, or
1.  remove the reserved namespace declaration.

Sample Error Message:

{{.CodeMode}}
{{index .Examples 0}}
{{.CodeMode}}
`

func init() {
	register(ReservedDirectoryNameErrorCode, reservedDirectoryNameErrorExamples, reservedDirectoryNameExplanation)
}

// ReservedDirectoryNameError represents an illegal usage of a reserved name.
type ReservedDirectoryNameError struct {
	Dir nomospath.Relative
}

// Error implements error.
func (e ReservedDirectoryNameError) Error() string {
	return format(e,
		"Directories MUST NOT have reserved namespace names. Rename or remove directory:\n\n"+
			"path: %[1]s\n"+
			"name: %[2]s",
		e.Dir.RelativeSlashPath(), e.Dir.Base())
}

// Code implements Error
func (e ReservedDirectoryNameError) Code() string { return ReservedDirectoryNameErrorCode }
