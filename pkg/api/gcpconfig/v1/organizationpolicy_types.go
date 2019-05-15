package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// OrganizationPolicy is the Schema for the organizationpolicies API
// +k8s:openapi-gen=true
type OrganizationPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OrganizationPolicySpec `json:"spec"`
	Status Status                 `json:"status,omitempty"`
}

// Conditions returns the list of conditions of resource status.
// It implements the resource.GenericObject interface.
func (op *OrganizationPolicy) Conditions() []Condition {
	return op.Status.Conditions
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// OrganizationPolicyList contains a list of OrganizationPolicy
type OrganizationPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OrganizationPolicy `json:"items"`
}
