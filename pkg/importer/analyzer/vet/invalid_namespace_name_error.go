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
	status.AddExamples(InvalidNamespaceNameErrorCode, InvalidNamespaceNameError(
		ns,
		"foo",
	))
}

var invalidNamespaceNameErrorstatus = status.NewErrorBuilder(InvalidNamespaceNameErrorCode)

// InvalidNamespaceNameError reports that a Namespace has an invalid name.
func InvalidNamespaceNameError(resource id.Resource, expected string) status.Error {
	return invalidNamespaceNameErrorstatus.WithResources(resource).Errorf(
		"A %[1]s MUST declare `metadata.name` that matches the name of its directory.\n\n"+
			"expected metadata.name: %[2]s",
		node.Namespace, expected)
}
