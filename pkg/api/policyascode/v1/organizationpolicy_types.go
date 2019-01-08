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
	// hcl is vendored and will be used in a future CL.
	_ "github.com/hashicorp/hcl"
	// hcl/printer is vendored and will be used in a future CL.
	_ "github.com/hashicorp/hcl/hcl/printer"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// OrganizationPolicy is the Schema for the organizationpolicies API
// +k8s:openapi-gen=true
type OrganizationPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OrganizationPolicySpec   `json:"spec"`
	Status OrganizationPolicyStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// OrganizationPolicyList contains a list of OrganizationPolicy
type OrganizationPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OrganizationPolicy `json:"items"`
}

func init() {
	SchemeBuilder.Register(&OrganizationPolicy{}, &OrganizationPolicyList{})
}
