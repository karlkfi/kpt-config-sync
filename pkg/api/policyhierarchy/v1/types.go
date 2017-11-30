/*
Copyright 2017 The Kubernetes Authors.

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
	rbac_v1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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
	Roles         []rbac_v1.Role            `json:"roles"`
	RoleBindings  []rbac_v1.RoleBinding     `json:"roleBindings"`
	ResourceQuota core_v1.ResourceQuotaSpec `json:"resourceQuota"`
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

///////////////// Stolos Resource Quota //////////////////

// StolosResourceQuotaResouce contains the name of the lone StolosResourceQuota
// resource.
const StolosResourceQuotaResource = "stolosresourcequotas"

// Genclient directive for StolosResourceQuota
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// StolosResourceQuota represents a resource quota object set on a policyspace.
// This is needed as it will be controlled by a stolos resource quota controller
// which will monitor usage in all the descendants of that policyspace.
type StolosResourceQuota struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata. The Name field of the policy node must match the namespace name.
	// +optional
	metav1.ObjectMeta `json:"metadata"`

	// The actual object definition, per K8S object definition style.
	Spec StolosResourceQuotaSpec `json:"spec"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// StolosResourceQuotaList holds a list of stolos resource quotas
type StolosResourceQuotaList struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items is a list of stolos resource quotas.
	Items []StolosResourceQuota `json:"items"`
}

// StolosResourceQuotaSpec is equivalent to the payload of the core_v1.ResourceQuotaStatus
type StolosResourceQuotaSpec struct {
	Status core_v1.ResourceQuotaStatus `json:"status"`
}
