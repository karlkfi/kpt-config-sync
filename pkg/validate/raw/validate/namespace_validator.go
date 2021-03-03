package validate

import (
	"github.com/google/nomos/pkg/api/configmanagement"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	"k8s.io/apimachinery/pkg/util/validation"
)

// Namespace verifies that the given FileObject has a valid namespace according
// to the following rules:
// - if the object is a Namespace, it must not be `config-management-system`
// - if the object has a metadata namespace, it must be a valid k8s namespace
//   and it must not be in `config-management-system`
func Namespace(obj ast.FileObject) status.Error {
	if obj.GroupVersionKind().GroupKind() == kinds.Namespace().GroupKind() {
		return validateNamespace(obj)
	}
	return validateObjectNamespace(obj)
}

func validateNamespace(obj ast.FileObject) status.Error {
	if !isValidNamespace(obj.GetName()) {
		return nonhierarchical.InvalidNamespaceError(obj)
	}
	if configmanagement.IsControllerNamespace(obj.GetName()) {
		return nonhierarchical.IllegalNamespace(obj)
	}
	return nil
}

func validateObjectNamespace(obj ast.FileObject) status.Error {
	ns := obj.GetNamespace()
	if ns == "" {
		return nil
	}
	if !isValidNamespace(ns) {
		return nonhierarchical.InvalidNamespaceError(obj)
	}
	if configmanagement.IsControllerNamespace(ns) {
		return nonhierarchical.ObjectInIllegalNamespace(obj)
	}
	return nil
}

// isValidNamespace returns true if Kubernetes allows Namespaces with the name "name".
func isValidNamespace(name string) bool {
	// IsDNS1123Label is misleading as the Kubernetes requirements are more stringent than the specification.
	errs := validation.IsDNS1123Label(name)
	return len(errs) == 0
}
