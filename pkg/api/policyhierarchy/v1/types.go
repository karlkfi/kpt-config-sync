/*
Copyright 2017 The Nomos Authors.

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
	core_v1 "k8s.io/api/core/v1"
	extensions_v1beta1 "k8s.io/api/extensions/v1beta1"
	rbac_v1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// These comments must remain outside the package docstring.
// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ClusterPolicy is the top-level object for the policy node data definition.
//
// It holds a policy defined for a single org unit (namespace).
// +protobuf=true
type ClusterPolicy struct {
	metav1.TypeMeta `json:",inline"`

	// Standard object's metadata.
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// The actual object definition, per K8S object definition style.
	// +optional
	Spec ClusterPolicySpec `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`

	// The current status of the object, per K8S object definition style.
	// +optional
	Status ClusterPolicyStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
}

// ClusterPolicySpec defines the policies that will exist at the cluster level.
// +protobuf=true
type ClusterPolicySpec struct {
	// +optional
	ClusterRolesV1 []rbac_v1.ClusterRole `json:"clusterRolesV1,omitempty" protobuf:"bytes,1,rep,name=clusterRolesV1"`

	// +optional
	ClusterRoleBindingsV1 []rbac_v1.ClusterRoleBinding `json:"clusterRoleBindingsV1,omitempty" protobuf:"bytes,2,rep,name=clusterRoleBindingsV1"`

	// +optional
	PodSecurityPoliciesV1Beta1 []extensions_v1beta1.PodSecurityPolicy `json:"podSecurityPolicyV1Beta1,omitempty" protobuf:"bytes,3,rep,name=podSecurityPolicyV1Beta1"`

	// ImportToken indicates the version of the ClusterPolicy last imported from the source of truth.
	// +optional
	ImportToken string `json:"importToken,omitempty" protobuf:"bytes,4,opt,name=importToken"`

	// ImportTime is the timestamp of when the ClusterPolicy was updated by the Importer.
	// +optional
	ImportTime metav1.Time `json:"importTime,omitempty" protobuf:"bytes,5,opt,name=importTime"`
}

// ClusterPolicyStatus contains fields that define the status of a ClusterPolicy.
// +protobuf=true
type ClusterPolicyStatus struct {
	// SyncToken indicates the version of that policy that the Syncer last attempted to update from.
	// +optional
	SyncToken string `json:"syncToken,omitempty" protobuf:"bytes,1,opt,name=syncToken"`

	// SyncErrors contains any errors that occurred during the last attempt the Syncer made to update
	// resources from the ClusterPolicy specs. This field will be empty on success.
	// +optional
	SyncErrors []ClusterPolicySyncError `json:"syncErrors,omitempty" protobuf:"bytes,2,rep,name=syncErrors"`

	// SyncTime is the timestamp of when the policy resources were last updated by the Syncer.
	// +optional
	SyncTime metav1.Time `json:"syncTime,omitempty" protobuf:"bytes,3,opt,name=syncTime"`

	// SyncState is the current state of the policy resources (eg synced, stale, error).
	// +optional
	SyncState PolicySyncState `json:"syncState,omitempty" protobuf:"bytes,4,opt,name=syncState"`
}

// ClusterPolicySyncError represents a failed sync attempt for a ClusterPolicy.
// +protobuf=true
type ClusterPolicySyncError struct {
	// ResourceName is the name of the K8S resource that failed to sync.
	// +optional
	ResourceName string `json:"resourceName,omitempty" protobuf:"bytes,1,opt,name=resourceName"`

	// ResourceKind is the type of the K8S resource (from TypeMeta.Kind).
	// +optional
	ResourceKind string `json:"resourceKind,omitempty" protobuf:"bytes,2,opt,name=resourceKind"`

	// ResourceAPI is the API and version of the K8S resource (from TypeMeta.ApiVersion).
	// +optional
	ResourceAPI string `json:"resourceApi,omitempty" protobuf:"bytes,3,opt,name=resourceApi"`

	// ErrorMessage contains the error returned from API server when trying to sync.
	// +optional
	ErrorMessage string `json:"errorMessage,omitempty" protobuf:"bytes,4,opt,name=errorMessage"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ClusterPolicyList holds a list of cluster level policies, returned as response to a List
// call on the cluster policy hierarchy.
// +protobuf=true
type ClusterPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// +optional
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Items is a list of policy nodes that apply.
	Items []ClusterPolicy `json:"items" protobuf:"bytes,2,rep,name=items"`
}

// These comments must remain outside the package docstring.
// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PolicyNode is the top-level object for the policy node data definition.
//
// It holds a policy defined for a single org unit (namespace).
// +protobuf=true
type PolicyNode struct {
	metav1.TypeMeta `json:",inline"`

	// Standard object's metadata. The Name field of the policy node must match the namespace name.
	// +optional
	metav1.ObjectMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`

	// The actual object definition, per K8S object definition style.
	// +optional
	Spec PolicyNodeSpec `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`

	// The current status of the object, per K8S object definition style.
	// +optional
	Status PolicyNodeStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
}

// PolicyNodeSpec contains all the information about a policy linkage.
// +protobuf=true
type PolicyNodeSpec struct {
	// The type of the PolicyNode.
	Type PolicyNodeType `json:"type,omitempty" protobuf:"varint,1,opt,name=type"`

	// The parent org unit
	// +optional
	Parent string `json:"parent,omitempty" protobuf:"bytes,2,opt,name=parent"`

	// +optional
	RolesV1 []rbac_v1.Role `json:"rolesV1,omitempty" protobuf:"bytes,3,rep,name=rolesV1"`

	// +optional
	RoleBindingsV1 []rbac_v1.RoleBinding `json:"roleBindingsV1,omitempty" protobuf:"bytes,4,rep,name=roleBindingsV1"`

	// +optional
	ResourceQuotaV1 *core_v1.ResourceQuota `json:"resourceQuotaV1,omitempty" protobuf:"bytes,5,opt,name=resourceQuotaV1"`

	// ImportToken indicates the version of the PolicyNode last imported from the source of truth.
	// +optional
	ImportToken string `json:"importToken,omitempty" protobuf:"bytes,6,opt,name=importToken"`

	// ImportTime is the timestamp of when the PolicyNode was updated by the Importer.
	// +optional
	ImportTime metav1.Time `json:"importTime,omitempty" protobuf:"bytes,7,opt,name=importTime"`
}

// PolicyNodeStatus contains fields that define the status of a PolicyNode. The fields related to Syncer
// will never be populated for PolicySpaces since they are flattened down to child Namespaces.
// +protobuf=true
type PolicyNodeStatus struct {
	// SyncTokens is a map of policy name to token indicating the version of that policy that the
	// Syncer last attempted to update from. There will always be one entry for the PolicyNode itself
	// as well as one entry for each PolicyNode up its hierarchy.
	// +optional
	SyncTokens map[string]string `json:"syncTokens,omitempty" protobuf:"bytes,1,rep,name=syncTokens"`

	// SyncErrors contains any errors that occurred during the last attempt the Syncer made to update
	// resources from the PolicyNode specs. This field will be empty on success.
	// +optional
	SyncErrors []PolicyNodeSyncError `json:"syncErrors,omitempty" protobuf:"bytes,2,rep,name=syncErrors"`

	// SyncTime is the timestamp of when the policy resources were last updated by the Syncer.
	// +optional
	SyncTime metav1.Time `json:"syncTime,omitempty" protobuf:"bytes,3,opt,name=syncTime"`

	// SyncState is the current state of the policy resources (eg synced, stale, error).
	// +optional
	SyncState PolicySyncState `json:"syncState,omitempty" protobuf:"bytes,4,opt,name=syncState"`
}

// PolicyNodeSyncError represents a failed sync attempt for a PolicyNode.
// +protobuf=true
type PolicyNodeSyncError struct {
	// SourceName is the name of the PolicyNode where the resource is defined.
	// +optional
	SourceName string `json:"sourceName,omitempty" protobuf:"bytes,1,opt,name=sourceName"`

	// ResourceName is the name of the K8S resource that failed to sync.
	// +optional
	ResourceName string `json:"resourceName,omitempty" protobuf:"bytes,2,opt,name=resourceName"`

	// ResourceKind is the type of the K8S resource (from TypeMeta.Kind).
	// +optional
	ResourceKind string `json:"resourceKind,omitempty" protobuf:"bytes,3,opt,name=resourceKind"`

	// ResourceAPI is the API and version of the K8S resource (from TypeMeta.ApiVersion).
	// +optional
	ResourceAPI string `json:"resourceApi,omitempty" protobuf:"bytes,4,opt,name=resourceApi"`

	// ErrorMessage contains the error returned from API server when trying to sync.
	// +optional
	ErrorMessage string `json:"errorMessage,omitempty" protobuf:"bytes,5,opt,name=errorMessage"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PolicyNodeList holds a list of namespace policies, as response to a List
// call on the policy hierarchy API.
// +protobuf=true
type PolicyNodeList struct {
	metav1.TypeMeta `json:",inline"`

	// Standard object's metadata.
	// +optional
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Items is a list of policy nodes that apply.
	Items []PolicyNode `json:"items" protobuf:"bytes,2,rep,name=items"`
}

// AllPolicies holds all PolicyNodes and the ClusterPolicy.
type AllPolicies struct {
	// Map of names to PolicyNodes.
	// +optional
	PolicyNodes map[string]PolicyNode `protobuf:"bytes,1,rep,name=policyNodes"`
	// +optional
	ClusterPolicy *ClusterPolicy `protobuf:"bytes,2,opt,name=clusterPolicy"`
}

// These comments must remain outside the package docstring.
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
	// +optional
	Spec NamespaceSelectorSpec `json:"spec" protobuf:"bytes,2,opt,name=spec"`
}

// NamespaceSelectorSpec contains spec fields for NamespaceSelector.
// +protobuf=true
type NamespaceSelectorSpec struct {
	// Selects namespaces.
	// This field is NOT optional and follows standard
	// label selector semantics. An empty selector matches all namespaces.
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

// SyncDeclaration is used for configuring sync of generic resources.
type SyncDeclaration struct {
	metav1.TypeMeta `json:",inline"`

	// Standard object's metadata. The Name field of the policy node must match the namespace name.
	// +optional
	metav1.ObjectMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`

	// Spec is the standard spec field.
	Spec SyncDeclarationSpec `json:"spec"`

	// Status is the status for the sync declaration.
	Status SyncDeclarationStatus `json:"status"`
}

// SyncDeclarationSpec specifies the sync declaration which corresponds to an API Group and contained
// kinds and versions.
type SyncDeclarationSpec struct {
	Group string              `json:"group"` // group, eg nomos.dev
	Kinds SyncDeclarationKind `json:"kinds"`
}

// SyncDeclarationKind represents the spec for a Kind of object we are syncing.
type SyncDeclarationKind struct {
	// Kind is the string that represents the Kind for the object as given in TypeMeta.
	Kind string `json:"kind,omitempty"`
	// Versions indicates the versions that will be handled for the object of Group and Kind.
	Versions []SyncDeclarationVersion `json:"versions,omitempty"`
}

// SyncDeclarationVersion corresponds to a single version in a (group, kind)
type SyncDeclarationVersion struct {
	// Version indicates the version used for the API Group, for example v1 or v1beta1.
	Version string `json:"version"`

	// CompareFields is a list of fields to compare against.  This will default to ["spec"] which should
	// be sufficient for most obejcts, however, there are exceptions (RBAC) so these need to be declared.
	CompareFields []string `json:"compareFields,omitempty"`
}

// SyncDeclarationStatus represents the status for a sync declaration
type SyncDeclarationStatus struct {
	Syncing bool   `json:"syncing,omitempty"` // Syncer should set this to true when it has begun using the sync decl
	Error   string `json:"error,omitempty"`   // Any errors encountered for this declaration.
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SyncDeclarationList holds a list of SyncDeclaration resources.
type SyncDeclarationList struct {
	metav1.TypeMeta `json:",inline"`

	// Standard object's metadata.
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items is a list of sync declarations.
	Items []SyncDeclaration `json:"items"`
}
