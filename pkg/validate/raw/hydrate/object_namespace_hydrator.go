package hydrate

import (
	"github.com/google/nomos/pkg/api/configmanagement/v1/repo"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/validate/objects"
)

// ObjectNamespaces hydrates the given raw Objects by setting the metadata
// namespace field for objects that are located in a namespace directory but do
// not have their namespace specified yet.
func ObjectNamespaces(objs *objects.Raw) status.MultiError {
	namespaces := make(map[string]bool)
	for _, obj := range objs.Objects {
		if isValidHierarchicalNamespace(obj) {
			namespaces[obj.GetName()] = true
		}
	}
	for _, obj := range objs.Objects {
		if topLevelDir(obj) != repo.NamespacesDir {
			continue
		}
		// Namespaces and NamespaceSelectors are the only cluster-scoped objects
		// expected under the namespace/ directory, so we want to make sure we
		// don't accidentally assign them a namespace.
		gk := obj.GroupVersionKind().GroupKind()
		if gk == kinds.Namespace().GroupKind() || gk == kinds.NamespaceSelector().GroupKind() {
			continue
		}
		if obj.GetNamespace() == "" {
			dir := obj.Dir().Base()
			if namespaces[dir] {
				obj.SetNamespace(dir)
			}
		}
	}
	return nil
}

func isValidHierarchicalNamespace(obj ast.FileObject) bool {
	if obj.GroupVersionKind().GroupKind() != kinds.Namespace().GroupKind() {
		return false
	}
	if topLevelDir(obj) != repo.NamespacesDir {
		return false
	}
	return obj.GetName() == obj.Dir().Base()
}

func topLevelDir(obj ast.FileObject) string {
	sourcePath := obj.Relative.OSPath()
	return cmpath.RelativeSlash(sourcePath).Split()[0]
}
