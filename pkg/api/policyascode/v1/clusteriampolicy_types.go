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

// ClusterIAMPolicy is the Schema for the clusteriampolicies API
type ClusterIAMPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IAMPolicySpec   `json:"spec,omitempty"`
	Status IAMPolicyStatus `json:"status,omitempty"`
}

// ClusterIAMPolicyList contains a list of ClusterIAMPolicy
type ClusterIAMPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterIAMPolicy `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ClusterIAMPolicy{}, &ClusterIAMPolicyList{})
}
