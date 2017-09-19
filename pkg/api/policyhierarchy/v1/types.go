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

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PolicyNode is the top-level object for the policy node data definition.
//
// It holds a policy defined for a single org unit (namespace).
type PolicyNode struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// +optional
	metav1.ObjectMeta `json:"metadata"`

	// The actual object definition, per K8S object definition style.
	Spec PolicyNodeSpec `json:"spec"`
}

// PolicyNodeSpec contains all the information about a policy linkage.
type PolicyNodeSpec struct {
	// The name of the org unit or the namespace.
	Name string `json:"name"`

	// True for leaf namespaces where pods will actually be scheduled,
	// false for the parent org unit namespace where this policy is linked
	// to, but no containers should run
	WorkingNamespace bool `json:"workingNamespace"`

	// The parent org unit
	Parent string `json:"parent"`

	// The policies attached to that node
	Policies PolicyLists `json:"policies"`
}

// PolicyLists contains all the defined policies that are linked to a particular
// PolicyNode.
type PolicyLists struct {
	Roles          []rbac_v1.Role              `json:"roles"`
	RoleBindings   []rbac_v1.RoleBinding       `json:"roleBindings"`
	ResourceQuotas []core_v1.ResourceQuotaSpec `json:"resourceQuotas"`
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
