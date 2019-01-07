package veterrors

import (
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
)

// InvalidNamespaceNameErrorCode is the error code for InvalidNamespaceNameError
const InvalidNamespaceNameErrorCode = "1020"

var invalidNamespaceNameErrorExample = InvalidNamespaceNameError{
	ResourceID: &resourceID{
		source:           "namespaces/foo/namespace.yaml",
		name:             "bar",
		groupVersionKind: kinds.Namespace()},
	Expected: "foo",
}

var invalidNamespacesNameErrorExplanation = `
A Namespace Resource MUST have a metadata.name that matches the name of its
directory. To fix, correct the offending Namespace's metadata.name or its
directory.
`

func init() {
	register(InvalidNamespaceNameErrorCode, invalidNamespaceNameErrorExample, invalidNamespacesNameErrorExplanation)
}

// InvalidNamespaceNameError reports that a Namespace has an invalid name.
type InvalidNamespaceNameError struct {
	ResourceID
	Expected string
}

// Error implements error
func (e InvalidNamespaceNameError) Error() string {
	return format(e,
		"A %[1]s MUST declare metadata.name that matches the name of its directory.\n\n"+
			"%[2]s\n\n"+
			"expected metadata.name: %[3]s\n",
		ast.Namespace, printResourceID(e), e.Expected)
}

// Code implements Error
func (e InvalidNamespaceNameError) Code() string { return InvalidNamespaceNameErrorCode }
