package v1

import "github.com/google/nomos/pkg/api/configmanagement"

const (
	// ManagedByKey is the recommended Kubernetes label for marking a resource as managed by an
	// application.
	ManagedByKey = "app.kubernetes.io/managed-by"
	// ManagedByValue marks the resource as managed by Nomos.
	ManagedByValue = configmanagement.GroupName

	// HierarchyControllerDepthSuffix is a label suffix for hierarchical namespace depth.
	// See definition at http://bit.ly/k8s-hnc-design#heading=h.1wg2oqxxn6ka.
	HierarchyControllerDepthSuffix = ".tree.hnc.x-k8s.io/depth"

	// DepthLabelRootName is the depth label name for the root node "namespaces" in the hierarchy.
	DepthLabelRootName = "config-sync-root"
)

// SyncerLabels returns the set of Nomos labels that the syncer should manage.
func SyncerLabels() map[string]string {
	return map[string]string{
		ManagedByKey: ManagedByValue,
	}
}
