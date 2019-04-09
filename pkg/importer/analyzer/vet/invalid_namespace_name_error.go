package vet

import (
	"github.com/google/nomos/pkg/importer/analyzer/ast/node"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/status"
)

// InvalidNamespaceNameErrorCode is the error code for InvalidNamespaceNameError
const InvalidNamespaceNameErrorCode = "1020"

func init() {
	ns := namespace(cmpath.FromSlash("namespaces/foo/ns.yaml"))
	ns.MetaObject().SetName("bar")
	status.Register(InvalidNamespaceNameErrorCode, InvalidNamespaceNameError{
		Resource: ns,
		Expected: "foo",
	})
}

// InvalidNamespaceNameError reports that a Namespace has an invalid name.
type InvalidNamespaceNameError struct {
	id.Resource
	Expected string
}

var _ status.ResourceError = &InvalidNamespaceNameError{}

// Error implements error
func (e InvalidNamespaceNameError) Error() string {
	return status.Format(e,
		"A %[1]s MUST declare `metadata.name` that matches the name of its directory.\n\n"+
			"expected metadata.name: %[2]s",
		node.Namespace, e.Expected)
}

// Code implements Error
func (e InvalidNamespaceNameError) Code() string { return InvalidNamespaceNameErrorCode }

// Resources implements ResourceError
func (e InvalidNamespaceNameError) Resources() []id.Resource {
	return []id.Resource{e.Resource}
}
