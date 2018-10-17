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

// OrganizationPolicySpec defines the desired state of OrganizationPolicy
type OrganizationPolicySpec struct {
	Constraints       []OrganizationPolicyConstraint `json:"constraints"`
	ResourceReference ResourceReference              `json:"resourceReference"`
	ImportDetails     ImportDetails                  `json:"importDetails"`
}

// OrganizationPolicyStatus defines the observed state of OrganizationPolicy
type OrganizationPolicyStatus struct {
	SyncDetails SyncDetails `json:"syncDetails,omitempty"`
}

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

// OrganizationPolicyConstraint is the Schema for Constraints in OrganizationPolicySpec
type OrganizationPolicyConstraint struct {
	Constraint    string                          `json:"constraint"`
	ListPolicy    OrganizationPolicyListPolicy    `json:"listPolicy,omitempty"`
	BooleanPolicy OrganizationPolicyBooleanPolicy `json:"booleanPolicy,omitempty"`
}

// OrganizationPolicyListPolicy is the Schema for ListPolicy in OrganizationPolicyConstraint
type OrganizationPolicyListPolicy struct {
	// +kubebuilder:validation:Pattern=^((is|under):)?(organizations|folders|projects)/
	AllowedValues []string `json:"allowedValues,omitempty"`
	// +kubebuilder:validation:Pattern=^((is|under):)?(organizations|folders|projects)/
	DisallowedValues []string `json:"disallowedValues,omitempty"`
	// +kubebuilder:validation:Pattern=^(ALLOW|DENY)$
	AllValues         string `json:"allValues,omitempty"`
	InheritFromParent bool   `json:"inheritFromParent,omitempty"`
}

// OrganizationPolicyBooleanPolicy is the Schema for BooleanPolicy in OrganizationPolicyConstraint
type OrganizationPolicyBooleanPolicy struct {
	Enforced bool `json:"enforced"`
}

func init() {
	SchemeBuilder.Register(&OrganizationPolicy{}, &OrganizationPolicyList{})
}
