/*
Copyright 2017 The Nomos Authors.

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
// +protobuf=true
type ClusterPolicy struct {
	metav1.TypeMeta `json:",inline"`

	// Standard object's metadata.
	// +optional
	metav1.ObjectMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`

	// The actual object definition, per K8S object definition style.
	Spec ClusterPolicySpec `json:"spec" protobuf:"bytes,2,opt,name=spec"`
}

// ClusterPolicySpec defines the policies that will exist at the cluster level.
// +protobuf=true
type ClusterPolicySpec struct {
	// The policies specified for cluster level resources.
	Policies ClusterPolicies `json:"policies" protobuf:"bytes,1,opt,name=policies"`

	// UnmanagedNamespacesV1 is a configmap that contains unmanaged namespace configuration for the syncer.
	// Keys for the data field correspond to namespace names. The only value accepted at the moment is "unmanaged"
	// which indicates that the namespace is unmanaged.
	UnmanagedNamespacesV1 *core_v1.ConfigMap `json:"unmanagedNamespaces,omitempty" protobuf:"bytes,2,opt,name=unmanagedNamespaces"`
}

// ClusterPolicies specifies the policies nomos synchronizes to a cluster. This is factored out
// due to the fact that it is specified in MasterClusterPolicyNodeSpec and ClusterPolicyNodeSpec.
// +protobuf=true
type ClusterPolicies struct {
	// Cluster scope resources.
	ClusterRolesV1             []rbac_v1.ClusterRole                  `json:"clusterRolesV1" protobuf:"bytes,2,rep,name=clusterRolesV1"`
	ClusterRoleBindingsV1      []rbac_v1.ClusterRoleBinding           `json:"clusterRoleBindingsV1" protobuf:"bytes,3,rep,name=clusterRoleBindingsV1"`
	PodSecurityPoliciesV1Beta1 []extensions_v1beta1.PodSecurityPolicy `json:"podSecurityPolicyV1Beta1" protobuf:"bytes,4,rep,name=podSecurityPolicyV1Beta1"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ClusterPolicyList holds a list of cluster level policies, returned as response to a List
// call on the cluster policy hierarchy.
// +protobuf=true
type ClusterPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// +optional
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Items is a list of policy nodes that apply.
	Items []ClusterPolicy `json:"items" protobuf:"bytes,2,rep,name=items"`
}

// These comments must remain outside the package docstring.
// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PolicyNode is the top-level object for the policy node data definition.
//
// It holds a policy defined for a single org unit (namespace).
// +protobuf=true
type PolicyNode struct {
	metav1.TypeMeta `json:",inline"`

	// Standard object's metadata. The Name field of the policy node must match the namespace name.
	// +optional
	metav1.ObjectMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`

	// The actual object definition, per K8S object definition style.
	Spec PolicyNodeSpec `json:"spec" protobuf:"bytes,2,opt,name=spec"`
}

// PolicyNodeSpec contains all the information about a policy linkage.
// +protobuf=true
type PolicyNodeSpec struct {
	// The type of the PolicyNode.
	Type PolicyNodeType `json:"type,omitempty" protobuf:"varint,1,opt,name=type"`

	// The parent org unit
	Parent string `json:"parent" protobuf:"bytes,2,opt,name=parent"`

	// The policies attached to that node
	Policies Policies `json:"policies" protobuf:"bytes,3,opt,name=policies"`
}

// Policies contains all the defined policies that are linked to a particular
// PolicyNode.
// +protobuf=true
type Policies struct {
	RolesV1         []rbac_v1.Role         `json:"rolesV1" protobuf:"bytes,1,rep,name=rolesV1"`
	RoleBindingsV1  []rbac_v1.RoleBinding  `json:"roleBindingsV1" protobuf:"bytes,2,rep,name=roleBindingsV1"`
	ResourceQuotaV1 *core_v1.ResourceQuota `json:"resourceQuotaV1" protobuf:"bytes,3,opt,name=resourceQuotaV1"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PolicyNodeList holds a list of namespace policies, as response to a List
// call on the policy hierarchy API.
// +protobuf=true
type PolicyNodeList struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// +optional
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Items is a list of policy nodes that apply.
	Items []PolicyNode `json:"items" protobuf:"bytes,2,rep,name=items"`
}

// AllPolicies holds all PolicyNodes and the ClusterPolicy.
type AllPolicies struct {
	// Map of names to PolicyNodes.
	PolicyNodes   map[string]PolicyNode `protobuf:"bytes,1,rep,name=policyNodes"`
	ClusterPolicy *ClusterPolicy        `protobuf:"bytes,2,opt,name=clusterPolicy"`
}
