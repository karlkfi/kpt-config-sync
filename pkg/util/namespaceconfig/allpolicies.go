package namespaceconfig

import (
	v1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
)

// AllPolicies holds things that Importer wants to sync. It is only used in-process, not written
// directly as a Kubernetes resource.
type AllPolicies struct {
	// Map of names to NamespaceConfigs.
	// +optional
	NamespaceConfigs map[string]v1.NamespaceConfig `protobuf:"bytes,1,rep,name=namespaceConfigs"`
	// +optional
	ClusterConfig *v1.ClusterConfig `protobuf:"bytes,2,opt,name=clusterConfig"`
	// Map of names to Syncs.
	// +optional
	Syncs map[string]v1.Sync `protobuf:"bytes,3,rep,name=syncs"`
}
