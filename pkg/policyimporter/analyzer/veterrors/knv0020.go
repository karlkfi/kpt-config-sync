package veterrors

import (
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/id"
)

// InvalidNamespaceNameErrorCode is the error code for InvalidNamespaceNameError
const InvalidNamespaceNameErrorCode = "1020"

var invalidNamespaceNameErrorExamples = []Error{
	InvalidNamespaceNameError{
		Resource: &resourceID{
			source:           "namespaces/foo/namespace.yaml",
			name:             "bar",
			groupVersionKind: kinds.Namespace()},
		Expected: "foo",
	},
}

var invalidNamespacesNameErrorExplanation = `
A Namespace Resource MUST have a metadata.name that matches the name of its
directory. To fix, correct the offending Namespace's metadata.name or its
directory.

Sample Error Message:

{{.CodeMode}}
{{index .Examples 0}}
{{.CodeMode}}
`

func init() {
	register(InvalidNamespaceNameErrorCode, invalidNamespaceNameErrorExamples, invalidNamespacesNameErrorExplanation)
}

// InvalidNamespaceNameError reports that a Namespace has an invalid name.
type InvalidNamespaceNameError struct {
	id.Resource
	Expected string
}

// Error implements error
func (e InvalidNamespaceNameError) Error() string {
	return format(e,
		"A %[1]s MUST declare metadata.name that matches the name of its directory.\n\n"+
			"%[2]s\n\n"+
			"expected metadata.name: %[3]s\n",
		ast.Namespace, id.PrintResource(e), e.Expected)
}

// Code implements Error
func (e InvalidNamespaceNameError) Code() string { return InvalidNamespaceNameErrorCode }
