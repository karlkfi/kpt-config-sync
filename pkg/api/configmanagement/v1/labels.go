package v1

import "github.com/google/nomos/pkg/api/configmanagement"

const (
	// ManagedByKey is the recommended Kubernetes label for marking a resource as managed by an
	// application.
	ManagedByKey = "app.kubernetes.io/managed-by"
	// ManagedByValue marks the resource as managed by Nomos.
	ManagedByValue = configmanagement.GroupName
	// SystemLabel is the system Nomos label.
	SystemLabel = ConfigManagementPrefix + "system"
	// ArchLabel is the arch Nomos label.
	ArchLabel = ConfigManagementPrefix + "arch"
)

// SyncerLabels returns the set of Nomos labels that the syncer should manage.
func SyncerLabels() map[string]string {
	return map[string]string{
		ManagedByKey: ManagedByValue,
	}
}
