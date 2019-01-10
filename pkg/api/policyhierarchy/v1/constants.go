/*
Copyright 2018 The Nomos Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

// NoParentNamespace is the constant we use (empty string) for indicating that no parent exists
// for the policy node spec.  Only one policy node should have a parent with this value.
// This is also used as the value for the label set on a namespace.
const NoParentNamespace = ""

// ClusterPolicyName is the name of the singleton ClusterPolicy resource.
const ClusterPolicyName = "nomos-cluster-policy"

// RootPolicyNodeName is the name of the root PolicyNode object.
const RootPolicyNodeName = "nomos-root-node"

// PolicyNodeType represents the types of policynodes that can exist.
type PolicyNodeType string

// IsNamespace returns true if the type corresponds to a namespace type.
func (p PolicyNodeType) IsNamespace() bool {
	return p == Namespace
}

// IsPolicyspace returns true if the type corresponds to a policyspace type.
func (p PolicyNodeType) IsPolicyspace() bool {
	return p == Policyspace
}

const (
	// Policyspace indicates that the PolicyNode is for a Policyspace and should not be manifested
	// into a namespace
	Policyspace = PolicyNodeType("policyspace")

	// Namespace indicates that the PolicyNode is represents a Namespace that should be created
	// and managed on the cluster.
	Namespace = PolicyNodeType("namespace")
)

// PolicySyncState represents the states that a policynode or clusterpolicy can be in with regards
// to the source of truth.
type PolicySyncState string

// IsSynced returns true if the state indicates a policy that is synced to the source of truth.
func (p PolicySyncState) IsSynced() bool {
	return p == StateSynced
}

// IsUnknown returns true if the state is unknown or undeclared.
func (p PolicySyncState) IsUnknown() bool {
	return p == StateUnknown
}

const (
	// StateUnknown indicates that the policy's state is undeclared or unknown.
	StateUnknown = PolicySyncState("")

	// StateSynced indicates that the policy is the same as the last known version from the source of
	// truth.
	StateSynced = PolicySyncState("synced")

	// StateStale indicates that the policy is different than the last known version from the source
	// of truth.
	StateStale = PolicySyncState("stale")

	// StateError indicates that there was an error updating the policy to match the last known
	// version from the source of truth.
	StateError = PolicySyncState("error")
)
