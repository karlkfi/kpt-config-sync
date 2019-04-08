package vet

import (
	"github.com/google/nomos/pkg/api/configmanagement/v1/repo"
	"github.com/google/nomos/pkg/importer/analyzer/ast/node"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/status"
)

// IllegalTopLevelNamespaceErrorCode is the error code for IllegalTopLevelNamespaceError
const IllegalTopLevelNamespaceErrorCode = "1019"

func init() {
	status.Register(IllegalTopLevelNamespaceErrorCode, IllegalTopLevelNamespaceError{
		Resource: namespace(cmpath.FromSlash("namespaces/ns.yaml")),
	})
}

// IllegalTopLevelNamespaceError reports that there may not be a Namespace declared directly in namespaces/
type IllegalTopLevelNamespaceError struct {
	id.Resource
}

var _ status.ResourceError = &IllegalTopLevelNamespaceError{}

// Error implements error
func (e IllegalTopLevelNamespaceError) Error() string {
	return status.Format(e,
		"%[2]ss MUST be declared in subdirectories of %[1]s/. Create a subdirectory for %[2]ss declared in:\n\n"+
			"%[3]s",
		repo.NamespacesDir, node.Namespace, id.PrintResource(e))
}

// Code implements Error
func (e IllegalTopLevelNamespaceError) Code() string { return IllegalTopLevelNamespaceErrorCode }

// Resources implements ResourceError
func (e IllegalTopLevelNamespaceError) Resources() []id.Resource {
	return []id.Resource{e.Resource}
}
