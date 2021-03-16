package nonhierarchical

import (
	"github.com/google/nomos/pkg/api/configmanagement"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/status"
)

// IllegalNamespaceErrorCode is the error code for illegal Namespace definitions.
const IllegalNamespaceErrorCode = "1034"

var illegalNamespaceError = status.NewErrorBuilder(IllegalNamespaceErrorCode)

// ObjectInIllegalNamespace reports that an object has been declared in an illegal Namespace.
func ObjectInIllegalNamespace(resource id.Resource) status.Error {
	return illegalNamespaceError.
		Sprintf("Configs must not be declared in the %q namespace", configmanagement.ControllerNamespace).
		BuildWithResources(resource)
}

// IllegalNamespace reports that the config-management-system Namespace MUST NOT be declared.
func IllegalNamespace(resource id.Resource) status.Error {
	return illegalNamespaceError.
		Sprintf("The %q Namespace must not be declared", configmanagement.ControllerNamespace).
		BuildWithResources(resource)
}
