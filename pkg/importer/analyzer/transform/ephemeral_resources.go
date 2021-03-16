package transform

import (
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
