package resourcequota

// NamespaceTypeLabel is the label key used for nomos quotas.
const NamespaceTypeLabel = "nomos-namespace-type"
const (
	// NamespaceTypeWorkload is the value used for workload namespaces
	NamespaceTypeWorkload = "workload"
)

// ResourceQuotaObjectName is the resource name for quotas set by nomos.  We only allow one resource
// quota per namespace, so we hardcode the resource name.
const ResourceQuotaObjectName = "config-management-resource-quota"

// ResourceQuotaHierarchyName is the resource name for HierarchichalQuota.
const ResourceQuotaHierarchyName = "nomos-quota-hierarchy"

// ConfigManagementQuotaLabels are the labels applied to a workload namespace's quota object
var ConfigManagementQuotaLabels = NewConfigManagementQuotaLabels()

// NewConfigManagementQuotaLabels returns a new map of nomos quota labels since ConfigManagementQuotaLabels is mutable.
func NewConfigManagementQuotaLabels() map[string]string {
	return map[string]string{NamespaceTypeLabel: NamespaceTypeWorkload}
}
