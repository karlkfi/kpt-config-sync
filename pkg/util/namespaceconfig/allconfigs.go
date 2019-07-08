package namespaceconfig

import (
	"github.com/google/nomos/pkg/api/configmanagement/v1"
)

// AllConfigs holds things that Importer wants to sync. It is only used in-process, not written
// directly as a Kubernetes resource.
type AllConfigs struct {
	// Map of names to NamespaceConfigs.
	NamespaceConfigs map[string]v1.NamespaceConfig
	// Singleton config for non-CRD cluster-scoped resources.
	ClusterConfig *v1.ClusterConfig
	// Config with declared state for CRDs.
	CRDClusterConfig *v1.ClusterConfig
	// Map of names to Syncs.
	Syncs map[string]v1.Sync
}
