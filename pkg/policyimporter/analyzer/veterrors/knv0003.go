package veterrors

import "github.com/google/nomos/pkg/policyimporter/analyzer/ast"

// IllegalNamespaceSubdirectoryErrorCode is the error code for IllegalNamespaceSubdirectoryError
const IllegalNamespaceSubdirectoryErrorCode = "1003"

func init() {
	register(IllegalNamespaceSubdirectoryErrorCode, nil, "")
}

// IllegalNamespaceSubdirectoryError represents an illegal child directory of a namespace directory.
type IllegalNamespaceSubdirectoryError struct {
	Child  *ast.TreeNode
	Parent *ast.TreeNode
}

// Error implements error.
func (e IllegalNamespaceSubdirectoryError) Error() string {
	return format(e,
		"A %[1]s directory MUST NOT have subdirectories. "+
			"Restructure %[4]q so that it does not have subdirectory %[2]q:\n\n"+
			"%[3]s",
		ast.Namespace, e.Child.Name(), e.Child, e.Parent.Name())
}

// Code implements Error
func (e IllegalNamespaceSubdirectoryError) Code() string { return IllegalNamespaceSubdirectoryErrorCode }
