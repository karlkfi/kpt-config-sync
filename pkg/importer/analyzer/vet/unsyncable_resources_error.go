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
	status.Register(UnsyncableResourcesErrorCode, UnsyncableResourcesError{
		Dir: &ast.TreeNode{Path: cmpath.FromSlash("namespaces/foo/bar")},
	})
}

// UnsyncableResourcesError reports that an Abstract Namespace contains resources that cannot be synced because it has no
// Namespace directory descendants.
type UnsyncableResourcesError struct {
	Dir      id.TreeNode
	Ancestor bool
}

var _ status.PathError = &UnsyncableResourcesError{}

// Error implements error.
func (e UnsyncableResourcesError) Error() string {
	if e.Ancestor {
		return status.Format(e,
			"An %[1]s directory with configs MUST have at least one %[2]s subdirectory. "+
				"To fix, do one of the following: add a %[2]s directory below %[3]q, "+
				"convert a directory below to a %[2]s directory, "+
				"or remove the configs in %[3]q:\n\n"+
				"%[4]s",
			node.AbstractNamespace, node.Namespace, e.Dir.Name(), id.PrintTreeNode(e.Dir))
	}
	return status.Format(e,
		"An %[1]s directory with configs MUST have at least one %[2]s subdirectory. "+
			"To fix, do one of the following: add a %[2]s directory below %[3]q, "+
			"Add a Namespace config to %[3]q, "+
			"or remove the configs in %[3]q:\n\n"+
			"%[4]s",
		node.AbstractNamespace, node.Namespace, e.Dir.Name(), id.PrintTreeNode(e.Dir))
}

// Code implements Error
func (e UnsyncableResourcesError) Code() string { return UnsyncableResourcesErrorCode }

// RelativePaths implements PathError
func (e UnsyncableResourcesError) RelativePaths() []string {
	return []string{e.Dir.SlashPath()}
}
