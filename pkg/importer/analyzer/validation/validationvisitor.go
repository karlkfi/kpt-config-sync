package validation

import (
	"strings"

	"github.com/google/nomos/pkg/importer/analyzer/ast/node"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/status"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// IllegalNamespaceSubdirectoryErrorCode is the error code for IllegalNamespaceSubdirectoryError
const IllegalNamespaceSubdirectoryErrorCode = "1003"

var illegalNamespaceSubdirectoryError = status.NewErrorBuilder(IllegalNamespaceSubdirectoryErrorCode)

// IllegalNamespaceSubdirectoryError represents an illegal child directory of a namespace directory.
func IllegalNamespaceSubdirectoryError(child, parent id.TreeNode) status.Error {
	// TODO: We don't really need the parent node since it can be inferred from the Child.
	return illegalNamespaceSubdirectoryError.Sprintf("A %[1]s directory MUST NOT have subdirectories. "+
		"Restructure %[4]q so that it does not have subdirectory %[2]q:\n\n"+
		"%[3]s",
		node.Namespace, child.Name(), id.PrintTreeNode(child), parent.Name()).BuildWithPaths(child, parent)
}

// IllegalAbstractNamespaceObjectKindErrorCode is the error code for IllegalAbstractNamespaceObjectKindError
const IllegalAbstractNamespaceObjectKindErrorCode = "1007"

var illegalAbstractNamespaceObjectKindError = status.NewErrorBuilder(IllegalAbstractNamespaceObjectKindErrorCode)

// IllegalAbstractNamespaceObjectKindError represents an illegal usage of a kind not allowed in abstract namespaces.
// TODO(willbeason): Consolidate Illegal{X}ObjectKindErrors
func IllegalAbstractNamespaceObjectKindError(resource client.Object) status.Error {
	return illegalAbstractNamespaceObjectKindError.Sprintf(
		"Config %[3]q illegally declared in an %[1]s directory. "+
			"Move this config to a %[2]s directory:",
		strings.ToLower(string(node.AbstractNamespace)), strings.ToLower(string(node.Namespace)), resource.GetName()).
		BuildWithResources(resource)
}
