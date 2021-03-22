package nonhierarchical

import (
	"github.com/google/nomos/pkg/status"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ManagedResourceInUnmanagedNamespaceErrorCode is the error code for illegal
// managed resources in unmanaged Namespaces.
const ManagedResourceInUnmanagedNamespaceErrorCode = "1056"

var managedResourceInUnmanagedNamespaceError = status.NewErrorBuilder(ManagedResourceInUnmanagedNamespaceErrorCode)

// ManagedResourceInUnmanagedNamespace represents managed resources illegally
// declared in an unmanaged Namespace.
func ManagedResourceInUnmanagedNamespace(namespace string, resources ...client.Object) status.Error {
	return managedResourceInUnmanagedNamespaceError.
		Sprintf("Managed resources must not be declared in unmanaged Namespaces. Namespace %q is is declared unmanaged but contains managed resources. Either remove the managed: disabled annotation from Namespace %q or declare its resources as unmanaged by adding configmanagement.gke.io/managed:disabled annotation.", namespace, namespace).
		BuildWithResources(resources...)
}
