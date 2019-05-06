package vet

import (
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/ast/node"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/status"
)

// UnsyncableResourcesErrorCode is the error code for UnsyncableResourcesError
const UnsyncableResourcesErrorCode = "1044"

func init() {
	status.AddExamples(UnsyncableResourcesErrorCode,
		UnsyncableResourcesInLeaf(&ast.TreeNode{Path: cmpath.FromSlash("namespaces/foo/bar")}))
}

var unsyncableResourcesError = status.NewErrorBuilder(UnsyncableResourcesErrorCode)

// UnsyncableResourcesInLeaf reports that a leaf node has resources but is not a Namespace.
func UnsyncableResourcesInLeaf(dir id.TreeNode) status.Error {
	return unsyncableResourcesError.WithPaths(dir).Errorf(
		"An %[1]s directory with configs MUST have at least one %[2]s subdirectory. "+
			"To fix, do one of the following: add a %[2]s directory below %[3]q, "+
			"add a Namespace config to %[3]q, "+
			"or remove the configs in %[3]q:", node.AbstractNamespace, node.Namespace, dir.Name())
}

// UnsyncableResourcesInNonLeaf reports that a node has resources and descendants, but none of its
// descendants are Namespaces.
func UnsyncableResourcesInNonLeaf(dir id.TreeNode) status.Error {
	return unsyncableResourcesError.WithPaths(dir).Errorf(
		"An %[1]s directory with configs MUST have at least one %[2]s subdirectory. "+
			"To fix, do one of the following: add a %[2]s directory below %[3]q, "+
			"convert a directory below to a %[2]s directory, "+
			"or remove the configs in %[3]q:", node.AbstractNamespace, node.Namespace, dir.Name())
}
