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

// Package v1 has client stuff for our custom resource
// To generate clientset and deepcopy stuff (why aren't these done in one tool?) run:
// tools/generate-clientset.sh
package v1

import (
	core_v1 "k8s.io/api/core/v1"
	rbac_v1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PolicyNode holds a namespace policy
type PolicyNode struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// +optional
	metav1.ObjectMeta `json:"metadata"`
	Spec              PolicyNodeSpec `json:"spec"`
}

type PolicyNodeSpec struct {
	Name             string      `json:"name"`             // The name of the org unit or the namespace
	WorkingNamespace bool        `json:"workingNamespace"` // True for leaf namespaces where pods will actually be scheduled, false for the parent org unit namespace where policy is attached but no containers should run
	Parent           string      `json:"parent"`           // The parent org unit
	Policies         PolicyLists `json:"policies"`         // The policies attached to that node
}

type PolicyLists struct {
	Roles          []rbac_v1.Role              `json:"roles"`
	RoleBindings   []rbac_v1.RoleBinding       `json:"roleBindings"`
	ResourceQuotas []core_v1.ResourceQuotaSpec `json:"resourceQuotas"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PolicyNodeList holds a list of namespace policies
type PolicyNodeList struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items is a list of Roles
	Items []PolicyNode `json:"items"`
}
