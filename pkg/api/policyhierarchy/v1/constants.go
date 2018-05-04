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
const ParentLabelKey = "nomos-parent-ns"

// ClusterPolicyName is the name of the singleton ClusterPolicy resource.
const ClusterPolicyName = "nomos-cluster-policy"

// PolicyNodeType represents the types of policynodes that can exist.
type PolicyNodeType string

// IsNamespace returns true if the type corresponds to a namespace type.
func (p PolicyNodeType) IsNamespace() bool {
	return p == Namespace
}

// IsPolicyspace returns true if the type corresponds to a policyspaace type.
func (p PolicyNodeType) IsPolicyspace() bool {
	return p == Policyspace
}

// IsUnmanagedNamespace returns true if the type corresponds to an unmanaged namespace type.
func (p PolicyNodeType) IsUnmanagedNamespace() bool {
	return p == UnmanagedNamespace
}

const (
	// Policyspace indicates that the PolicyNode is for a Policyspace and should not be manifested
	// into a namespace
	Policyspace = PolicyNodeType("namespace")

	// Namespace indicates that the PolicyNode is represents a Namespace that should be created
	// and managed on the cluster.
	Namespace = PolicyNodeType("policyspace")

	// UnmanagedNamespace indicates that the namespace's policies will not be managed by nomos but
	// nomos will ensure the namespace exists.
	UnmanagedNamespace = PolicyNodeType("unmanagedNamespace")
)
