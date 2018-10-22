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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// These comments must remain outside the package docstring.
// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ClusterSelector specifies a LabelSelector applied to clusters that exist within a
// cluster registry.
//
// +protobuf=true
type ClusterSelector struct {
	metav1.TypeMeta `json:",inline"`

	// Standard object's metadata.
	// +optional
	metav1.ObjectMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`

	// The actual object definition, per K8S object definition style.
	Spec ClusterSelectorSpec `json:"spec" protobuf:"bytes,2,opt,name=spec"`
}

// ClusterSelectorSpec contains spec fields for ClusterSelector.
// +protobuf=true
type ClusterSelectorSpec struct {
	// Selects clusters.
	// This field is NOT optional and follows standard label selector semantics. An empty selector
	// matches all clusters.
	Selector metav1.LabelSelector `json:"selector" protobuf:"bytes,1,opt,name=selector"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ClusterSelectorList holds a list of ClusterSelector resources.
// +protobuf=true
type ClusterSelectorList struct {
	metav1.TypeMeta `json:",inline"`

	// Standard object's metadata.
	// +optional
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Items is a list of selectors.
	Items []ClusterSelector `json:"items" protobuf:"bytes,2,rep,name=items"`
}

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NamespaceSelector specifies a LabelSelector applied to namespaces that exist within a
// PolicyNode hierarchy.
//
// +protobuf=true
type NamespaceSelector struct {
	metav1.TypeMeta `json:",inline"`

	// Standard object's metadata. The Name field of the policy node must match the namespace name.
	// +optional
	metav1.ObjectMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`

	// The actual object definition, per K8S object definition style.
	Spec NamespaceSelectorSpec `json:"spec" protobuf:"bytes,2,opt,name=spec"`
}

// NamespaceSelectorSpec contains spec fields for NamespaceSelector.
// +protobuf=true
type NamespaceSelectorSpec struct {
	// Selects namespaces.
	// This field is NOT optional and follows standard label selector semantics. An empty selector
	// matches all namespaces.
	Selector metav1.LabelSelector `json:"selector" protobuf:"bytes,1,opt,name=selector"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NamespaceSelectorList holds a list of NamespaceSelector resources.
// +protobuf=true
type NamespaceSelectorList struct {
	metav1.TypeMeta `json:",inline"`

	// Standard object's metadata.
	// +optional
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Items is a list of policy nodes that apply.
	Items []NamespaceSelector `json:"items" protobuf:"bytes,2,rep,name=items"`
}

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Sync is used for configuring sync of generic resources.
type Sync struct {
	metav1.TypeMeta `json:",inline"`

	// Standard object's metadata. The Name field of the policy node must match the namespace name.
	// +optional
	metav1.ObjectMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`

	// Spec is the standard spec field.
	Spec SyncSpec `json:"spec"`

	// Status is the status for the sync declaration.
	Status SyncStatus `json:"status,omitempty"`
}

// SyncSpec specifies the sync declaration which corresponds to an API Group and contained
// kinds and versions.
type SyncSpec struct {
	// Groups represents all groups that are declared for sync.
	Groups []SyncGroup `json:"groups"` // groups, eg nomos.dev
}

// SyncGroup represents sync declarations for a Group.
type SyncGroup struct {
	// Group is the group, for example nomos.dev or rbac.authorization.k8s.io
	Group string `json:"group"` // group, eg nomos.dev
	// Kinds represents kinds from the Group.
	Kinds []SyncKind `json:"kinds"`
}

// SyncKind represents the spec for a Kind of object we are syncing.
type SyncKind struct {
	// Kind is the string that represents the Kind for the object as given in TypeMeta, for example
	// ClusterRole, Namespace or Deployment.
	Kind string `json:"kind"`
	// Mode indicates the sync mode for the object.
	// +optional
	Mode SyncMode `json:"mode,omitempty"`
	// Versions indicates the versions that will be handled for the object of Group and Kind.
	Versions []SyncVersion `json:"versions"`
}

// SyncVersion corresponds to a single version in a (group, kind)
type SyncVersion struct {
	// Version indicates the version used for the API Group, for example v1, v1beta1, v1alpha1.
	Version string `json:"version"`

	// CompareFields is an optional list of top-level fields to compare against.  This will default to
	// ["spec"] which should be sufficient for most objects, however, there are exceptions (RBAC) so
	// these need to be declared.
	// In addition to "spec" or the the fields declared here, we also compare annotations and labels.
	// Anything else stored in metadata is ignored for comparison purposes.
	CompareFields []string `json:"compareFields,omitempty"`
}

// SyncStatus represents the status for a sync declaration
type SyncStatus struct {
	GroupVersionKinds []SyncGroupVersionKindStatus `json:"groupVersionKinds,omitempty"`
}

// SyncGroupVersionKindStatus is a per Group, Kind status for the sync state of a resource.
type SyncGroupVersionKindStatus struct {
	// Group is the group, for example nomos.dev or rbac.authorization.k8s.io
	Group string `json:"group"`
	// Version is the version.
	Version string `json:"version"`
	// Kind is the string that represents the Kind for the object as given in TypeMeta, for exmple
	// ClusterRole, Namespace or Deployment.
	Kind string `json:"kind"`
	// Status indicates the state of the sync.  One of "syncing", or "error".  If "error" is specified
	// then Error will be populated with a message regarding the error.
	Status SyncState `json:"status"`
	// Message indicates a message associated with the status.
	Message string `json:"error,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SyncList holds a list of Sync resources.
type SyncList struct {
	metav1.TypeMeta `json:",inline"`

	// Standard object's metadata.
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items is a list of sync declarations.
	Items []Sync `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NomosConfig holds configuration for Nomos itself.
type NomosConfig struct {
	metav1.TypeMeta `json:",inline"`

	// Standard object's metadata.
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Spec NomosConfigSpec `json:"spec,omitempty"`
}

// NomosConfigSpec contains spec fields for NomosConfig.
type NomosConfigSpec struct {
	// Repo version string, corresponds to how policy importer should handle the
	// directory structure (implicit assumptions).
	RepoVersion string `json:"repoVersion"`
}
