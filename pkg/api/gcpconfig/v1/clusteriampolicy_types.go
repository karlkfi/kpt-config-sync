package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient:nonNamespaced

// ClusterIAMPolicy is the Schema for the clusteriampolicies API
// +k8s:openapi-gen=true
type ClusterIAMPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IAMPolicySpec `json:"spec,omitempty"`
	Status Status        `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ClusterIAMPolicyList contains a list of ClusterIAMPolicy
type ClusterIAMPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterIAMPolicy `json:"items"`
}

// Conditions returns the list of conditions of resource status.
// It implements the resource.GenericObject interface.
func (i *ClusterIAMPolicy) Conditions() []Condition {
	return i.Status.Conditions
}
