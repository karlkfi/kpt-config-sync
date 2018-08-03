/*
Copyright 2018 The Nomos Authors.

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

// TODO(cflewis): These types are marked "Prototype" as they are imported from
// policyhierarchy as part of setting up the policyascode directory structure.
// They will eventually be removed and replaced with the required policyascode
// types.

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// These comments must remain outside the function docstring.
// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// +protobuf=true
type PrototypePolicyAsCode struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	// +optional
	Spec PrototypePolicyAsCodeSpec `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`
}

// +protobuf=true
type PrototypePolicyAsCodeSpec struct {
}

// These comments must remain outside the function docstring.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// +protobuf=true
type PrototypePolicyAsCodeList struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// +optional
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	// Items is a list of PrototypePolicyAsCode.
	Items []PrototypePolicyAsCode `json:"items" protobuf:"bytes,2,rep,name=items"`
}
