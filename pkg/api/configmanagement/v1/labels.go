package v1

import "github.com/google/nomos/pkg/api/configmanagement"

const (
	// ConfigManagementSystemKey is the label indicating the resource is part of the Nomos installation.
	ConfigManagementSystemKey = ConfigManagementPrefix + "system"
	// ConfigManagementSystemValue marks that the resource is part of the system.
	ConfigManagementSystemValue = "true"

	// ConfigManagementQuotaKey is the label indicating whether hierarchical quota applies to a
	// Namespace.
	ConfigManagementQuotaKey = ConfigManagementPrefix + "quota"
	// ConfigManagementQuotaValue marks that quota is enabled for the Namespace.
	ConfigManagementQuotaValue = "true"

	// ManagedByKey is the recommended Kubernetes label for marking a resource as managed by an
	// application.
	ManagedByKey = "app.kubernetes.io/managed-by"
	// ManagedByValue marks the resource as managed by Nomos.
	ManagedByValue = configmanagement.GroupName
)

// SyncerLabels returns the set of Nomos labels that the syncer should manage.
func SyncerLabels() []string {
	return []string{
		ConfigManagementQuotaKey,
		ManagedByKey,
	}
}
