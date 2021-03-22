package validate

import (
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// UnmanagedNamespaces verifies that no managed resources are located in
// unmanaged Namespaces.
func UnmanagedNamespaces(objs []ast.FileObject) status.MultiError {
	unmanagedNamespaces := make(map[string][]client.Object)
	for _, obj := range objs {
		if obj.GetObjectKind().GroupVersionKind() != kinds.Namespace() {
			continue
		}
		if isUnmanaged(obj) {
			unmanagedNamespaces[obj.GetName()] = []client.Object{}
		}
	}

	for _, obj := range objs {
		ns := obj.GetNamespace()
		if ns == "" || isUnmanaged(obj) {
			continue
		}
		resources, isInUnmanagedNamespace := unmanagedNamespaces[ns]
		if isInUnmanagedNamespace {
			unmanagedNamespaces[ns] = append(resources, obj)
		}
	}

	var errs status.MultiError
	for ns, resources := range unmanagedNamespaces {
		if len(resources) > 0 {
			errs = status.Append(errs, nonhierarchical.ManagedResourceInUnmanagedNamespace(ns, resources...))
		}
	}
	return errs
}

func isUnmanaged(obj client.Object) bool {
	annotation, hasAnnotation := obj.GetAnnotations()[v1.ResourceManagementKey]
	return hasAnnotation && annotation == v1.ResourceManagementDisabled
}
