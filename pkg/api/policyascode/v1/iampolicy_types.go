/*
Copyright 2018 Google LLC.

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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// IAMPolicySpec defines the desired state of IAMPolicy
type IAMPolicySpec struct {
	ResourceReference ResourceReference  `json:"resourceReference"`
	Bindings          []IAMPolicyBinding `json:"bindings"`
	ImportDetails     ImportDetails      `json:"importDetails"`
}

// IAMPolicyBinding is the Schema for Bindings in IAMPolicySpec
type IAMPolicyBinding struct {
	// +kubebuilder:validation:Pattern=^(user|serviceAccount|group|domain):.+$
	Members []string `json:"members"`
	// +kubebuilder:validation:Pattern=^roles/[\w\.]+$
	Role string `json:"role"`
}

// IAMPolicyStatus defines the observed state of IAMPolicy
type IAMPolicyStatus struct {
	SyncDetails SyncDetails `json:"syncDetails,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// IAMPolicy is the Schema for the iampolicies API
// +k8s:openapi-gen=true
type IAMPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IAMPolicySpec   `json:"spec"`
	Status IAMPolicyStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// IAMPolicyList contains a list of IAMPolicy
type IAMPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IAMPolicy `json:"items"`
}

func init() {
	SchemeBuilder.Register(&IAMPolicy{}, &IAMPolicyList{})
}
