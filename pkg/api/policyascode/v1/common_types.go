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

// ImportDetails defines detailed import information for CRD operations.
// Not consolidating ImportDetails and SyncDetails on token and time because
// in ImportDetails they are required, while in SyncDetails they are not.
type ImportDetails struct {
	// +kubebuilder:validation:Pattern=^\w{40}$
	Token string `json:"token"`
	// +kubebuilder:validation:Format=dateTime
	Time metav1.Time `json:"time"`
}

// SyncDetails defines detailed sync information for CRD operations
type SyncDetails struct {
	// +kubebuilder:validation:Pattern=^\w{40}$
	Token string `json:"token,omitempty"`
	// +kubebuilder:validation:Format=dateTime
	Time  metav1.Time `json:"time,omitempty"`
	Error string      `json:"error,omitempty"`
}

// ParentReference defines schema to denote parent resource. Note that
// ParentReference and ResourceReference are no consolidated, because
// ParentReference cannot be "Project".
type ParentReference struct {
	// +kubebuilder:validation:Pattern=^(Organization|Folder)$
	Kind string `json:"kind"`
	Name string `json:"name"`
}

// ResourceReference defines schema to denote resource
type ResourceReference struct {
	// +kubebuilder:validation:Pattern=^(Organization|Folder|Project)$
	Kind string `json:"kind"`
	Name string `json:"name"`
}

// IAMPolicyBinding is the Schema for Bindings in IAMPolicySpec
type IAMPolicyBinding struct {
	// +kubebuilder:validation:Pattern=^(user|serviceAccount|group|domain):.+$
	Members []string `json:"members"`
	// +kubebuilder:validation:Pattern=^roles/[\w\.]+$
	Role string `json:"role"`
}

// IAMPolicySpec defines the desired state of IAMPolicy
type IAMPolicySpec struct {
	ResourceReference ResourceReference  `json:"resourceReference"`
	Bindings          []IAMPolicyBinding `json:"bindings"`
	ImportDetails     ImportDetails      `json:"importDetails"`
}

// IAMPolicyStatus defines the observed state of IAMPolicy
type IAMPolicyStatus struct {
	SyncDetails SyncDetails `json:"syncDetails,omitempty"`
}
