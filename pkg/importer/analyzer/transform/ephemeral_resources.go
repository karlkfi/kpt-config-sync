package transform

import (
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/kinds"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// IsEphemeral returns true if the type should not be synced to the cluster.
func IsEphemeral(gvk schema.GroupVersionKind) bool {
	return gvk == kinds.NamespaceSelector() ||
		gvk == kinds.Sync() ||
		gvk == kinds.Repo() ||
		gvk == kinds.HierarchyConfig()
}

// RemoveEphemeralResources removes resources that were needed before for
// processing, but may now be safely discarded.
//
// Ideally the last logic needing these resources should discard them.
func RemoveEphemeralResources(objects []ast.FileObject) []ast.FileObject {
	var result []ast.FileObject
	for _, o := range objects {
		if !IsEphemeral(o.GroupVersionKind()) {
			result = append(result, o)
		}
	}
	return result
}
