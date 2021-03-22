package nonhierarchical

import (
	"github.com/google/nomos/pkg/api/configmanagement"
	"github.com/google/nomos/pkg/status"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// IllegalNamespaceErrorCode is the error code for illegal Namespace definitions.
const IllegalNamespaceErrorCode = "1034"

var illegalNamespaceError = status.NewErrorBuilder(IllegalNamespaceErrorCode)

// ObjectInIllegalNamespace reports that an object has been declared in an illegal Namespace.
func ObjectInIllegalNamespace(resource client.Object) status.Error {
	return illegalNamespaceError.
		Sprintf("Configs must not be declared in the %q namespace", configmanagement.ControllerNamespace).
		BuildWithResources(resource)
}

// IllegalNamespace reports that the config-management-system Namespace MUST NOT be declared.
func IllegalNamespace(resource client.Object) status.Error {
	return illegalNamespaceError.
		Sprintf("The %q Namespace must not be declared", configmanagement.ControllerNamespace).
		BuildWithResources(resource)
}
