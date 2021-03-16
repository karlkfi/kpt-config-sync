package semantic

import (
	"github.com/google/nomos/pkg/importer/analyzer/ast/node"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/status"
)

// UnsyncableResourcesErrorCode is the error code for UnsyncableResourcesError
const UnsyncableResourcesErrorCode = "1044"

var unsyncableResourcesError = status.NewErrorBuilder(UnsyncableResourcesErrorCode)

// UnsyncableResourcesInLeaf reports that a leaf node has resources but is not a Namespace.
func UnsyncableResourcesInLeaf(dir id.TreeNode) status.Error {
	return unsyncableResourcesError.
		Sprintf("The directory %[2]q has configs, but is missing a %[1]s "+
			"config. All bottom level subdirectories MUST have a %[1]s config.", node.Namespace, dir.Name()).
		BuildWithPaths(dir)
}

// UnsyncableResourcesInNonLeaf reports that a node has resources and descendants, but none of its
// descendants are Namespaces.
func UnsyncableResourcesInNonLeaf(dir id.TreeNode) status.Error {
	return unsyncableResourcesError.
		Sprintf("The %[1]s directory named %[3]q has resources and "+
			"subdirectories, but none of its subdirectories are Namespaces. An %[1]s"+
			" MUST have at least one %[2]s subdirectory.", node.AbstractNamespace, node.Namespace, dir.Name()).
		BuildWithPaths(dir)
}
