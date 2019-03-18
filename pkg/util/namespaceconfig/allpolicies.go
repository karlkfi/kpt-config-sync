package namespaceconfig

import (
	v1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
)

// AllPolicies holds things that Importer wants to sync. It is only used in-process, not written
// directly as a Kubernetes resource.
type AllPolicies struct {
	// Map of names to NamespaceConfigs.
	NamespaceConfigs map[string]v1.NamespaceConfig
	// Singleton config for the cluster.
	ClusterConfig *v1.ClusterConfig
	// Map of names to Syncs.
	Syncs map[string]v1.Sync
	// Singleton Repo for the cluster.
	Repo *v1.Repo
}
