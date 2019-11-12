package syntax

import (
	"strings"

	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/visitor"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/util/namespaceutil"
)

// NewDirectoryNameValidator validates that directory names are valid and not reserved.
func NewDirectoryNameValidator() *visitor.ValidatorVisitor {
	return visitor.NewTreeNodeValidator(
		func(n *ast.TreeNode) status.MultiError {
			name := n.Base()
			if namespaceutil.IsInvalid(name) {
				return InvalidDirectoryNameError(n.Path)
			} else if namespaceutil.IsReserved(name) {
				return ReservedDirectoryNameError(n.Path)
			}
			return nil
		})
}

// ReservedDirectoryNameErrorCode is the error code for ReservedDirectoryNameError
const ReservedDirectoryNameErrorCode = "1001"

var reservedDirectoryNameError = status.NewErrorBuilder(ReservedDirectoryNameErrorCode)

// ReservedDirectoryNameError represents an illegal usage of a reserved name.
func ReservedDirectoryNameError(dir id.Path) status.Error {
	return reservedDirectoryNameError.Errorf("Directories MUST NOT have reserved namespace names. Rename or remove %q:",
		dir.OSPath())
}

// InvalidDirectoryNameErrorCode is the error code for InvalidDirectoryNameError
const InvalidDirectoryNameErrorCode = "1028"

var invalidDirectoryNameError = status.NewErrorBuilder(InvalidDirectoryNameErrorCode)

// InvalidDirectoryNameError represents an illegal usage of a reserved name.
func InvalidDirectoryNameError(dir cmpath.Path) status.Error {
	return invalidDirectoryNameError.WithPaths(dir).Errorf(
		"Directory names must have fewer than 64 characters, consist of lower case alphanumeric characters or '-', and must "+
			"start and end with an alphanumeric character. Rename or remove the %q directory:", dir.Base())
}

// InvalidNamespaceError reports using an illegal Namespace.
func InvalidNamespaceError(o id.Resource, errs []string) status.Error {
	return invalidDirectoryNameError.WithResources(o).Errorf(
		"metadata.namespace is invalid:\n\n%s\n", strings.Join(errs, "\n"))
}
