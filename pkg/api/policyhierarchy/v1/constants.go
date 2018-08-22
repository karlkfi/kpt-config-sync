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

// ParentLabelKey is the Key of a label set on a namespace with value set to the parent namespace's
// name.
const ParentLabelKey = "nomos.dev/parent-name"

// NamespaceSelectorAnnotationKey is the annotation key set on policy resources that refers to
// name of NamespaceSelector resource.
const NamespaceSelectorAnnotationKey = "nomos.dev/namespace-selector"

// ClusterPolicyName is the name of the singleton ClusterPolicy resource.
const ClusterPolicyName = "nomos-cluster-policy"

// ReservedNamespacesConfigMapName is the name of the ConfigMap specifying reserved namespaces.
const ReservedNamespacesConfigMapName = "nomos-reserved-namespaces"

// NamespaceAttribute is an attribute defining how Nomos reacts to reserved namespaces.
type NamespaceAttribute string

const (
	// ReservedAttribute means that these namespaces will not be managed by Nomos.
	ReservedAttribute NamespaceAttribute = "reserved"
)

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

// IsReservedNamespace returns true if the type corresponds to a reserved namespace type.
func (p PolicyNodeType) IsReservedNamespace() bool {
	return p == ReservedNamespace
}

const (
	// Policyspace indicates that the PolicyNode is for a Policyspace and should not be manifested
	// into a namespace
	Policyspace = PolicyNodeType("policyspace")

	// Namespace indicates that the PolicyNode is represents a Namespace that should be created
	// and managed on the cluster.
	Namespace = PolicyNodeType("namespace")

	// ReservedNamespace indicates that the namespace's policies will not be managed by Nomos.
	ReservedNamespace = PolicyNodeType("reservedNamespace")
)

// PolicySyncState represents the states that a policynode or clusterpolicy can be in with regards
// to the source of truth.
type PolicySyncState string

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
