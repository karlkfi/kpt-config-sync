package metadata

import (
	"github.com/google/nomos/pkg/api/configmanagement"
	"github.com/google/nomos/pkg/api/configsync"
)

// Labels with the `configmanagement.gke.io/` prefix.
const (
	// ManagedByValue marks the resource as managed by Nomos.
	ManagedByValue = configmanagement.GroupName
	// SystemLabel is the system Nomos label.
	SystemLabel = ConfigManagementPrefix + "system"
	// ArchLabel is the arch Nomos label.
	ArchLabel = ConfigManagementPrefix + "arch"
)

// Labels with the `configsync.gke.io/` prefix.
const (
	// ReconcilerLabel is the unique label given to each reconciler pod.
	// This label is set by Config Sync on a root-reconciler or namespace-reconciler pod.
	ReconcilerLabel = configsync.ConfigSyncPrefix + "reconciler"

	// DeclaredVersionLabel declares the API Version in which a resource was initially
	// declared.
	// This label is set by Config Sync on a managed resource.
	DeclaredVersionLabel = configsync.ConfigSyncPrefix + "declared-version"
)

// DepthSuffix is a label suffix for hierarchical namespace depth.
// See definition at http://bit.ly/k8s-hnc-design#heading=h.1wg2oqxxn6ka.
// This label is set by Config Sync on a managed namespace resource.
const DepthSuffix = ".tree.hnc.x-k8s.io/depth"

// ManagedByKey is the recommended Kubernetes label for marking a resource as managed by an
// application.
const ManagedByKey = "app.kubernetes.io/managed-by"

// SyncerLabels returns the Nomos labels that the syncer should manage.
func SyncerLabels() map[string]string {
	return map[string]string{
		ManagedByKey: ManagedByValue,
	}
}
