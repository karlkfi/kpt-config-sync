/*
Copyright 2017 The Stolos Authors.

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

import (
	core_v1 "k8s.io/api/core/v1"
	extensions_v1beta1 "k8s.io/api/extensions/v1beta1"
	rbac_v1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// These comments must remain outside the package docstring.
// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ClusterPolicy is the top-level object for the policy node data definition.
//
// It holds a policy defined for a single org unit (namespace).
type ClusterPolicy struct {
	metav1.TypeMeta `json:",inline"`

	// Standard object's metadata.
	// +optional
	metav1.ObjectMeta `json:"metadata"`

	// The actual object definition, per K8S object definition style.
	Spec ClusterPolicySpec `json:"spec"`
}

// ClusterPolicySpec defines the policies that will exist at the cluster level.
type ClusterPolicySpec struct {
	// Sources describes the resource / name / resourceVersion of definitions that were merged to
	// create this object, for example ["clusterpolicy.prod.275564"]. Note that there is no ambiguity
	// in this as the resource name and resource version are not allowed to contain the '.' character.
	// This field will not be set in the MasterPolicyNode and will only be set at enrolled clusters.
	Sources []string `json:"sources"`

	// The policies specified for cluster level resources.
	Policies ClusterPolicies `json:"policies"`
}

const (
	// ClusterPolicyClusterRoles is the name of the ClusterPolicy resource that will store cluster roles
	ClusterPolicyClusterRoles = "clusterrole"
	// ClusterPolicyClusterRoleBindings is the name of the ClusterPolicy resource that will store cluster role bindings
	ClusterPolicyClusterRoleBindings = "clusterrolebinding"
	// ClusterPolicyPodSecurityPolicies is the name of the ClusterPolicy resource that will store pod security policies
	ClusterPolicyPodSecurityPolicies = "podsecuritypolicy"
)

// ClusterPolicies specifies the policies stolos synchronizes to a cluster. This is factored out
// due to the fact that it is specified in MasterClusterPolicyNodeSpec and ClusterPolicyNodeSpec.
type ClusterPolicies struct {
	// Type defines the type of resources that this holds. It will hold one of the cluster scoped
	// resources and should have a resource name that matches the resource type it holds.
	Type string `json:"type"`

	// Cluster scope resources.
	ClusterRolesV1             []rbac_v1.ClusterRole                  `json:"clusterRolesV1"`
	ClusterRoleBindingsV1      []rbac_v1.ClusterRoleBinding           `json:"clusterRoleBindingsV1"`
	PodSecurtiyPoliciesV1Beta1 []extensions_v1beta1.PodSecurityPolicy `json:"podSecurityPolicyV1Beta1"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ClusterPolicyList holds a list of cluster level policies, returned as response to a List
// call on the cluster policy hierarchy.
type ClusterPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items is a list of policy nodes that apply.
	Items []ClusterPolicy `json:"items"`
}

// These comments must remain outside the package docstring.
// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PolicyNode is the top-level object for the policy node data definition.
//
// It holds a policy defined for a single org unit (namespace).
type PolicyNode struct {
	metav1.TypeMeta `json:",inline"`

	// Standard object's metadata. The Name field of the policy node must match the namespace name.
	// +optional
	metav1.ObjectMeta `json:"metadata"`

	// The actual object definition, per K8S object definition style.
	Spec PolicyNodeSpec `json:"spec"`
}

// NoParentNamespace is the constant we use (empty string) for indicating that no parent exists
// for the policy node spec.  Only one policy node should have a parent with this value.
// This is also used as the value for the label set on a namespace.
const NoParentNamespace string = ""

// Key of a label set on a namespace with value set to the parent namespace's name.
const ParentLabelKey = "stolos-parent-ns"

// PolicyNodeSpec contains all the information about a policy linkage.
type PolicyNodeSpec struct {
	// False for leaf namespaces where pods will actually be scheduled,
	// True for the parent org unit namespace where this policy is linked
	// to, but no containers should run
	Policyspace bool `json:"policyspace"`

	// The parent org unit
	Parent string `json:"parent"`

	// The policies attached to that node
	Policies Policies `json:"policies"`
}

// Policies contains all the defined policies that are linked to a particular
// PolicyNode.
type Policies struct {
	RolesV1         []rbac_v1.Role            `json:"rolesV1"`
	RoleBindingsV1  []rbac_v1.RoleBinding     `json:"roleBindingsV1"`
	ResourceQuotaV1 core_v1.ResourceQuotaSpec `json:"resourceQuotaV1"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PolicyNodeList holds a list of namespace policies, as response to a List
// call on the policy hierarchy API.
type PolicyNodeList struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items is a list of policy nodes that apply.
	Items []PolicyNode `json:"items"`
}

// AllPolicies holds all PolicyNodes and the ClusterPolicy.
type AllPolicies struct {
	// Map of names to PolicyNodes.
	PolicyNodes   map[string]PolicyNode
	ClusterPolicy *ClusterPolicy
}
