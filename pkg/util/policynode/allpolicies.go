package policynode

import (
	v1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
)

// AllPolicies holds things that Importer wants to sync. It is only used in-process, not written
// directly as a Kubernetes resource.
type AllPolicies struct {
	// Map of names to PolicyNodes.
	// +optional
	PolicyNodes map[string]v1.PolicyNode `protobuf:"bytes,1,rep,name=policyNodes"`
	// +optional
	ClusterPolicy *v1.ClusterPolicy `protobuf:"bytes,2,opt,name=clusterPolicy"`
	// Map of names to Syncs.
	// +optional
	Syncs map[string]v1.Sync `protobuf:"bytes,3,rep,name=syncs"`
}
