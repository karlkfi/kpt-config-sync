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
	status.AddExamples(IllegalTopLevelNamespaceErrorCode, IllegalTopLevelNamespaceError(
		namespace(cmpath.FromSlash("namespaces/ns.yaml")),
	))
}

var illegalTopLevelNamespaceError = status.NewErrorBuilder(IllegalTopLevelNamespaceErrorCode)

// IllegalTopLevelNamespaceError reports that there may not be a Namespace declared directly in namespaces/
// Error implements error
func IllegalTopLevelNamespaceError(resource id.Resource) status.Error {
	return illegalTopLevelNamespaceError.Errorf(
		"%[2]ss MUST be declared in subdirectories of %[1]s/. Create a subdirectory for %[2]ss declared in:",
		repo.NamespacesDir, node.Namespace)
}
