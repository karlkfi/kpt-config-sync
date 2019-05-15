package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient:nonNamespaced

// ClusterOrganizationPolicy is the Schema for the clusterorganizationpolicies API
// +k8s:openapi-gen=true
type ClusterOrganizationPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OrganizationPolicySpec `json:"spec,omitempty"`
	Status Status                 `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ClusterOrganizationPolicyList contains a list of ClusterOrganizationPolicy
type ClusterOrganizationPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterOrganizationPolicy `json:"items"`
}

// Conditions returns the list of conditions of resource status.
// It implements the resource.GenericObject interface.
func (cop *ClusterOrganizationPolicy) Conditions() []Condition {
	return cop.Status.Conditions
}
