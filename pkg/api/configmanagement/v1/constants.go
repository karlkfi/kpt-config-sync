package v1

import "github.com/google/nomos/pkg/api/configmanagement"

// ClusterConfigName is the name of the ClusterConfig for all non-CRD cluster resources.
const ClusterConfigName = "config-management-cluster-config"

// CRDClusterConfigName is the name of the ClusterConfig for CRD resources.
const CRDClusterConfigName = "config-management-crd-cluster-config"

// ConfigSyncState represents the states that a namespaceconfig or clusterconfig can be in with regards
// to the source of truth.
type ConfigSyncState string

// IsSynced returns true if the state indicates a config that is synced to the source of truth.
func (p ConfigSyncState) IsSynced() bool {
	return p == StateSynced
}

// IsUnknown returns true if the state is unknown or undeclared.
func (p ConfigSyncState) IsUnknown() bool {
	return p == StateUnknown
}

const (
	// StateUnknown indicates that the config's state is undeclared or unknown.
	StateUnknown = ConfigSyncState("")

	// StateSynced indicates that the config is the same as the last known version from the source of
	// truth.
	StateSynced = ConfigSyncState("synced")

	// StateStale indicates that the config is different than the last known version from the source
	// of truth.
	StateStale = ConfigSyncState("stale")

	// StateError indicates that there was an error updating the config to match the last known
	// version from the source of truth.
	StateError = ConfigSyncState("error")
)

// SyncState indicates the state of a sync for resources of a particular group and kind.
type SyncState string

const (
	// Syncing indicates these resources are being actively managed by Nomos.
	Syncing SyncState = "syncing"
)

// SyncFinalizer is a finalizer handled by Syncer to ensure Sync deletions complete before Importer writes ClusterConfig
// and NamespaceConfig resources.
const SyncFinalizer = "syncer." + configmanagement.GroupName

// HierarchyModeType defines hierarchical behavior for namespaced objects.
type HierarchyModeType string

const (
	// HierarchyModeHierarchicalQuota indicates special aggregation behavior for ResourceQuota. With
	// this option, the config is copied to namespaces, but it is also left in the abstract namespace.
	// There, the ResourceQuotaAdmissionController uses the value to enforce the ResourceQuota in
	// aggregate for all descendent namespaces.
	//
	// This mode can only be used for ResourceQuota.
	HierarchyModeHierarchicalQuota = HierarchyModeType("hierarchicalQuota")
	// HierarchyModeInherit indicates that the resource can appear in abstract namespace directories
	// and will be inherited by any descendent namespaces. Without this value on the Sync, resources
	// must not appear in abstract namespaces.
	HierarchyModeInherit = HierarchyModeType("inherit")
	// HierarchyModeNone indicates that the resource cannot appear in abstract namespace directories.
	// For most resource types, this is the same as default, and it's not necessary to specify this
	// value. But RoleBinding and ResourceQuota have different default behaviors, and this value is
	// used to disable inheritance behaviors for those types.
	HierarchyModeNone = HierarchyModeType("none")
	// HierarchyModeDefault is the default value. Default behavior is type-specific.
	HierarchyModeDefault = HierarchyModeType("")
)

// HierarchyNodeType represents the types of hierarchical nodes that can exist.
type HierarchyNodeType string

const (
	// HierarchyNodeNamespace indicates that the node represents a namespace.
	HierarchyNodeNamespace = HierarchyNodeType("namespace")
	// HierarchyNodeAbstractNamespace indicates that the node represents an abstract namespace.
	HierarchyNodeAbstractNamespace = HierarchyNodeType("abstractNamespace")
)

// NoParentNamespace is the constant we use (empty string) for indicating that no parent exists
// for the hierarchy node.  Only one hierarchy node node should have a parent with this value.
const NoParentNamespace = ""
