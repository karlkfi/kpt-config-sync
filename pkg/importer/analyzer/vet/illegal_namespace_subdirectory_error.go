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
	status.Register(IllegalNamespaceSubdirectoryErrorCode, IllegalNamespaceSubdirectoryError{
		Child:  &ast.TreeNode{Path: cmpath.FromSlash("namespaces/foo/bar")},
		Parent: &ast.TreeNode{Path: cmpath.FromSlash("namespaces/foo")},
	})
}

// IllegalNamespaceSubdirectoryError represents an illegal child directory of a namespace directory.
type IllegalNamespaceSubdirectoryError struct {
	Child id.TreeNode
	// TODO: We don't really need the parent node since it can be inferred from the Child.
	Parent id.TreeNode
}

var _ status.PathError = &IllegalNamespaceSubdirectoryError{}

// Error implements error.
func (e IllegalNamespaceSubdirectoryError) Error() string {
	return status.Format(e,
		"A %[1]s directory MUST NOT have subdirectories. "+
			"Restructure %[4]q so that it does not have subdirectory %[2]q:\n\n"+
			"%[3]s",
		node.Namespace, e.Child.Name(), id.PrintTreeNode(e.Child), e.Parent.Name())
}

// Code implements Error
func (e IllegalNamespaceSubdirectoryError) Code() string { return IllegalNamespaceSubdirectoryErrorCode }

// RelativePaths implements PathError
func (e IllegalNamespaceSubdirectoryError) RelativePaths() []string {
	return []string{e.Child.SlashPath(), e.Parent.SlashPath()}
}
