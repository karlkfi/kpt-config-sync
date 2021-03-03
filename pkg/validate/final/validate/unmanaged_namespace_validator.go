package validate

import (
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
)

// UnmanagedNamespaces verifies that no managed resources are located in
// unmanaged Namespaces.
func UnmanagedNamespaces(objs []ast.FileObject) status.MultiError {
	unmanagedNamespaces := make(map[string][]id.Resource)
	for _, obj := range objs {
		if obj.GroupVersionKind() != kinds.Namespace() {
			continue
		}
		if isUnmanaged(obj) {
			unmanagedNamespaces[obj.GetName()] = []id.Resource{}
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

func isUnmanaged(obj core.LabeledAndAnnotated) bool {
	annotation, hasAnnotation := obj.GetAnnotations()[v1.ResourceManagementKey]
	return hasAnnotation && annotation == v1.ResourceManagementDisabled
}
