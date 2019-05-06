package vet

import (
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/ast/node"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/status"
)

// IllegalNamespaceSubdirectoryErrorCode is the error code for IllegalNamespaceSubdirectoryError
const IllegalNamespaceSubdirectoryErrorCode = "1003"

func init() {
	status.AddExamples(IllegalNamespaceSubdirectoryErrorCode, IllegalNamespaceSubdirectoryError(
		&ast.TreeNode{Path: cmpath.FromSlash("namespaces/foo/bar")},
		&ast.TreeNode{Path: cmpath.FromSlash("namespaces/foo")}))
}

var illegalNamespaceSubdirectoryError = status.NewErrorBuilder(IllegalNamespaceSubdirectoryErrorCode)

// IllegalNamespaceSubdirectoryError represents an illegal child directory of a namespace directory.
func IllegalNamespaceSubdirectoryError(child, parent id.TreeNode) status.Error {
	// TODO: We don't really need the parent node since it can be inferred from the Child.
	return illegalNamespaceSubdirectoryError.WithPaths(child, parent).Errorf("A %[1]s directory MUST NOT have subdirectories. "+
		"Restructure %[4]q so that it does not have subdirectory %[2]q:\n\n"+
		"%[3]s",
		node.Namespace, child.Name(), id.PrintTreeNode(child), parent.Name())
}
