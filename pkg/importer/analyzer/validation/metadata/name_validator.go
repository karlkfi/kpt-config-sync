package metadata

import (
	"github.com/google/nomos/pkg/api/configmanagement/v1/repo"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/ast/node"
	"github.com/google/nomos/pkg/importer/analyzer/visitor"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
)

// NewNamespaceDirectoryNameValidator validates the value of metadata.name is correct for Namespaces.
func NewNamespaceDirectoryNameValidator() *visitor.ValidatorVisitor {
	return visitor.NewAllObjectValidator(func(o ast.FileObject) status.MultiError {
		if o.GroupVersionKind() != kinds.Namespace() {
			return nil
		}

		expectedName := o.Dir().Base()
		if expectedName == repo.NamespacesDir {
			return IllegalTopLevelNamespaceError(&o)
		}
		if o.GetName() != expectedName {
			return InvalidNamespaceNameError(&o, expectedName)
		}
		return nil
	})
}

// IllegalTopLevelNamespaceErrorCode is the error code for IllegalTopLevelNamespaceError
const IllegalTopLevelNamespaceErrorCode = "1019"

var illegalTopLevelNamespaceError = status.NewErrorBuilder(IllegalTopLevelNamespaceErrorCode)

// IllegalTopLevelNamespaceError reports that there may not be a Namespace declared directly in namespaces/
// Error implements error
func IllegalTopLevelNamespaceError(resource id.Resource) status.Error {
	return illegalTopLevelNamespaceError.
		Sprintf("%[2]ss MUST be declared in subdirectories of '%[1]s/'. Create a subdirectory for the following %[2]s configs:",
			repo.NamespacesDir, node.Namespace).
		BuildWithResources(resource)
}

// InvalidNamespaceNameErrorCode is the error code for InvalidNamespaceNameError
const InvalidNamespaceNameErrorCode = "1020"

var invalidNamespaceNameErrorBuilder = status.NewErrorBuilder(InvalidNamespaceNameErrorCode)

// InvalidNamespaceNameError reports that a Namespace has an invalid name.
func InvalidNamespaceNameError(resource id.Resource, expected string) status.Error {
	return invalidNamespaceNameErrorBuilder.
		Sprintf("A %[1]s MUST declare `metadata.name` that matches the name of its directory.\n\n"+
			"expected `metadata.name`: %[2]s",
			node.Namespace, expected).
		BuildWithResources(resource)
}
