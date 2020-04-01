package nonhierarchical

import (
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
)

// ManagedResourceInUnmanagedNamespaceErrorCode is the error code for illegal
// managed resources in unmanaged Namespaces.
const ManagedResourceInUnmanagedNamespaceErrorCode = "1056"

var managedResourceInUnmanagedNamespaceError = status.NewErrorBuilder(ManagedResourceInUnmanagedNamespaceErrorCode)

// ManagedResourceInUnmanagedNamespace represents managed resources illegally
// declared in an unmanaged Namespace.
func ManagedResourceInUnmanagedNamespace(namespace string, resources ...id.Resource) status.Error {
	return managedResourceInUnmanagedNamespaceError.
		Sprintf("Managed resources must not be declared in unmanaged Namespaces. Namespace %q is is declared unmanaged but contains managed resources. Either remove the managed: disabled annotation from Namespace %q or declare its resources as unmanaged by adding configmanagement.gke.io/managed:disabled annotation.", namespace, namespace).
		BuildWithResources(resources...)
}

func isUnmanaged(o core.Annotated) bool {
	annotation, hasAnnotation := o.GetAnnotations()[v1.ResourceManagementKey]
	return hasAnnotation && annotation == v1.ResourceManagementDisabled
}

// ManagedNamespaceValidator reports errors when it detects managed resources
// in unmanaged Namespaces.
var ManagedNamespaceValidator = validator{
	validate: func(objects []ast.FileObject) status.MultiError {
		unmanagedNamespaces := make(map[string][]id.Resource)
		for _, o := range objects {
			if o.GroupVersionKind() != kinds.Namespace() {
				continue
			}
			if isUnmanaged(o) {
				unmanagedNamespaces[o.GetName()] = []id.Resource{}
			}
		}

		for _, o := range objects {
			ns := o.GetNamespace()
			if ns == "" {
				continue
			}
			if isUnmanaged(o) {
				continue
			}
			_, isInUnmanagedNamespace := unmanagedNamespaces[o.GetNamespace()]
			if isInUnmanagedNamespace {
				unmanagedNamespaces[o.GetNamespace()] = append(unmanagedNamespaces[o.GetNamespace()], o)
			}
		}

		var errs status.MultiError
		for ns, os := range unmanagedNamespaces {
			if len(os) > 0 {
				errs = status.Append(errs, ManagedResourceInUnmanagedNamespace(ns, os...))
			}
		}
		return errs
	},
}
