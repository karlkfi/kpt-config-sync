package syntax

import (
	"strings"

	"github.com/google/nomos/pkg/api/configmanagement"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/visitor"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/status"
	"k8s.io/apimachinery/pkg/util/validation"
)

// NewDirectoryNameValidator validates that directory names are valid and not reserved.
func NewDirectoryNameValidator() ast.Visitor {
	return visitor.NewTreeNodeValidator(
		func(n *ast.TreeNode) status.MultiError {
			name := n.Base()
			if !isValid(name) {
				return InvalidDirectoryNameError(n.Path)
			}
			if configmanagement.IsControllerNamespace(name) {
				return ReservedDirectoryNameError(n.Path)
			}
			return nil
		})
}

// isValid returns true if Kubernetes allows Namespaces with the name "name".
func isValid(name string) bool {
	// IsDNS1123Label is misleading as the Kubernetes requirements are more stringent than the specification.
	errs := validation.IsDNS1123Label(name)
	return len(errs) == 0
}

// InvalidDirectoryNameErrorCode is the error code for InvalidDirectoryNameError
const InvalidDirectoryNameErrorCode = "1028"

var invalidDirectoryNameError = status.NewErrorBuilder(InvalidDirectoryNameErrorCode)

// ReservedDirectoryNameError represents an illegal usage of a reserved name.
func ReservedDirectoryNameError(dir cmpath.Path) status.Error {
	// TODO(willbeason): Consider moving to Namespace validation instead.
	//  Strictly speaking, having a directory named "config-management-system" doesn't necessarily mean there are
	//  any resources declared in that Namespace. That would make this error message clearer.
	return invalidDirectoryNameError.WithPaths(dir).
		Errorf("%s repositories MUST NOT declare configs in the %s Namespace. Rename or remove the %q directory.",
			configmanagement.ProductName, configmanagement.ControllerNamespace, dir.Base())
}

// InvalidDirectoryNameError represents an illegal usage of a reserved name.
func InvalidDirectoryNameError(dir cmpath.Path) status.Error {
	return invalidDirectoryNameError.WithPaths(dir).Errorf(
		`Directory names MUST be valid Kubernetes Namespace names. Rename %q so that it:
1. has a length of 63 characters or fewer;
2. consists only of lowercase letters (a-z), digits (0-9), and hyphen '-'; and
3. begins and ends with a lowercase letter or digit.`, dir.Base())
}

// InvalidNamespaceError reports using an illegal Namespace.
func InvalidNamespaceError(o id.Resource, errs []string) status.Error {
	return invalidDirectoryNameError.WithResources(o).Errorf(
		"metadata.namespace is invalid:\n\n%s\n", strings.Join(errs, "\n"))
}
