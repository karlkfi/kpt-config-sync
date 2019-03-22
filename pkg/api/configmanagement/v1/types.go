/*
Copyright 2017 The CSP Config Management Authors.

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
	"fmt"
	"strings"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// These comments must remain outside the package docstring.
// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ClusterConfig is the top-level object for the policy node data definition.
//
// It holds a policy defined for a single org unit (namespace).
// +protobuf=true
type ClusterConfig struct {
	metav1.TypeMeta `json:",inline"`

	// Standard object's metadata.
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// The actual object definition, per K8S object definition style.
	// +optional
	Spec ClusterConfigSpec `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`

	// The current status of the object, per K8S object definition style.
	// +optional
	Status ClusterConfigStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
}

// ClusterConfigSpec defines the policies that will exist at the cluster level.
// +protobuf=true
type ClusterConfigSpec struct {
	// ImportToken indicates the version of the ClusterConfig last imported from the source of truth.
	// +optional
	ImportToken string `json:"importToken,omitempty" protobuf:"bytes,4,opt,name=importToken"`

	// ImportTime is the timestamp of when the ClusterConfig was updated by the Importer.
	// +optional
	ImportTime metav1.Time `json:"importTime,omitempty" protobuf:"bytes,5,opt,name=importTime"`

	// Resources contains namespace scoped resources that are synced to the API server.
	// +optional
	Resources []GenericResources `json:"resources,omitempty" protobuf:"bytes,8,opt,name=resources"`
}

// ClusterConfigStatus contains fields that define the status of a ClusterConfig.
// +protobuf=true
type ClusterConfigStatus struct {
	// SyncToken indicates the version of that policy that the Syncer last attempted to update from.
	// +optional
	SyncToken string `json:"syncToken,omitempty" protobuf:"bytes,1,opt,name=syncToken"`

	// SyncErrors contains any errors that occurred during the last attempt the Syncer made to update
	// resources from the ClusterConfig specs. This field will be empty on success.
	// +optional
	SyncErrors []ClusterConfigSyncError `json:"syncErrors,omitempty" protobuf:"bytes,2,rep,name=syncErrors"`

	// SyncTime is the timestamp of when the policy resources were last updated by the Syncer.
	// +optional
	SyncTime metav1.Time `json:"syncTime,omitempty" protobuf:"bytes,3,opt,name=syncTime"`

	// SyncState is the current state of the policy resources (eg synced, stale, error).
	// +optional
	SyncState PolicySyncState `json:"syncState,omitempty" protobuf:"bytes,4,opt,name=syncState"`
}

// ClusterConfigSyncError represents a failed sync attempt for a ClusterConfig.
// +protobuf=true
type ClusterConfigSyncError struct {
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

// ClusterConfigList holds a list of cluster level policies, returned as response to a List
// call on the cluster policy hierarchy.
// +protobuf=true
type ClusterConfigList struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// +optional
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Items is a list of policy nodes that apply.
	Items []ClusterConfig `json:"items" protobuf:"bytes,2,rep,name=items"`
}

// These comments must remain outside the package docstring.
// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NamespaceConfig is the top-level object for the policy node data definition.
//
// It holds a policy defined for a single org unit (namespace).
// +protobuf=true
type NamespaceConfig struct {
	metav1.TypeMeta `json:",inline"`

	// Standard object's metadata. The Name field of the policy node must match the namespace name.
	// +optional
	metav1.ObjectMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`

	// The actual object definition, per K8S object definition style.
	// +optional
	Spec NamespaceConfigSpec `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`

	// The current status of the object, per K8S object definition style.
	// +optional
	Status NamespaceConfigStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
}

// NamespaceConfigSpec contains all the information about a policy linkage.
// +protobuf=true
type NamespaceConfigSpec struct {

	// ImportToken indicates the version of the NamespaceConfig last imported from the source of truth.
	// +optional
	ImportToken string `json:"importToken,omitempty" protobuf:"bytes,6,opt,name=importToken"`

	// ImportTime is the timestamp of when the NamespaceConfig was updated by the Importer.
	// +optional
	ImportTime metav1.Time `json:"importTime,omitempty" protobuf:"bytes,7,opt,name=importTime"`

	// Resources contains namespace scoped resources that are synced to the API server.
	// +optional
	Resources []GenericResources `json:"resources,omitempty" protobuf:"bytes,8,opt,name=resources"`
}

// NamespaceConfigStatus contains fields that define the status of a NamespaceConfig.
// +protobuf=true
type NamespaceConfigStatus struct {
	// SyncToken indicates the version of that policy that the Syncer last attempted to update from.
	// +optional
	SyncToken string `json:"syncToken,omitempty" protobuf:"bytes,1,rep,name=syncToken"`

	// SyncErrors contains any errors that occurred during the last attempt the Syncer made to update
	// resources from the NamespaceConfig specs. This field will be empty on success.
	// +optional
	SyncErrors []NamespaceConfigSyncError `json:"syncErrors,omitempty" protobuf:"bytes,2,rep,name=syncErrors"`

	// SyncTime is the timestamp of when the policy resources were last updated by the Syncer.
	// +optional
	SyncTime metav1.Time `json:"syncTime,omitempty" protobuf:"bytes,3,opt,name=syncTime"`

	// SyncState is the current state of the policy resources (eg synced, stale, error).
	// +optional
	SyncState PolicySyncState `json:"syncState,omitempty" protobuf:"bytes,4,opt,name=syncState"`
}

// NamespaceConfigSyncError represents a failed sync attempt for a NamespaceConfig.
// +protobuf=true
type NamespaceConfigSyncError struct {
	// SourceName is the name of the NamespaceConfig where the resource is defined.
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

// NamespaceConfigList holds a list of namespace policies, as response to a List
// call on the policy hierarchy API.
// +protobuf=true
type NamespaceConfigList struct {
	metav1.TypeMeta `json:",inline"`

	// Standard object's metadata.
	// +optional
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Items is a list of policy nodes that apply.
	Items []NamespaceConfig `json:"items" protobuf:"bytes,2,rep,name=items"`
}

// GenericResources contains API objects of a specified Group and Kind.
// +protobuf=true
type GenericResources struct {
	// Group is the Group for all resources contained within
	// +optional
	Group string `json:"group,omitempty" protobuf:"bytes,1,opt,name=group"`

	// Kind is the Kind for all resoruces contained within.
	Kind string `json:"kind" protobuf:"bytes,2,opt,name=kind"`

	// Versions is a list Versions corresponding to the Version for this Group and Kind.
	Versions []GenericVersionResources `json:"versions" protobuf:"bytes,3,opt,name=versions"` // Per version information.
}

// GenericVersionResources holds a set of resources of a single version for a Group and Kind.
// +protobuf=true
type GenericVersionResources struct {
	// Version is the version of all objects in Objects.
	Version string `json:"version" protobuf:"bytes,1,opt,name=version"`

	// Objects is the list of objects of a single Group Version and Kind.
	Objects []runtime.RawExtension `json:"objects" protobuf:"bytes,2,opt,name=object"`
}

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
// NamespaceConfig hierarchy.
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

	// Items is a list of NamespaceSelectors.
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

// NewSync creates a sync object for consumption by the syncer, this will only populate the
// group and kind as those are the only fields the syncer presently consumes.
func NewSync(group, kind string) *Sync {
	var name string
	if group == "" {
		name = strings.ToLower(kind)
	} else {
		name = fmt.Sprintf("%s.%s", strings.ToLower(kind), group)
	}
	return &Sync{
		TypeMeta: metav1.TypeMeta{
			APIVersion: SchemeGroupVersion.String(),
			Kind:       "Sync",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: SyncSpec{
			Group: group,
			Kind:  kind,
		},
	}
}

// SyncSpec specifies the sync declaration which corresponds to an API Group and contained
// kinds and versions.
type SyncSpec struct {
	// Group is the group, for example configmanagement.gke.io or rbac.authorization.k8s.io
	Group string `json:"group"` // group, eg configmanagement.gke.io
	// Kind is the string that represents the Kind for the object as given in TypeMeta, for example
	// ClusterRole, Namespace or Deployment.
	Kind string `json:"kind"`
	// HierarchyMode specifies how the object is treated when it appears in an abstract namespace.
	// The default is off, meaning objects cannot appear in an abstract namespace. For RoleBinding,
	// the default is "inherit". For ResourceQuota, the default is "hierarchicalQuota".
	// +optional
	HierarchyMode HierarchyModeType `json:"hierarchyMode,omitempty"`
}

// SyncStatus represents the status for a sync declaration
type SyncStatus struct {
	// Status indicates the state of the sync.  One of "syncing", or "error".  If "error" is specified
	// then Error will be populated with a message regarding the error.
	Status SyncState `json:"status"`
	// Message indicates a message associated with the status.
	// +optional
	Message string `json:"message,omitempty"`
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

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Repo holds configuration and status about the Nomos source of truth.
// +protobuf=true
type Repo struct {
	metav1.TypeMeta `json:",inline"`

	// Standard object's metadata.
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// +optional
	Spec RepoSpec `json:"spec,omitempty"`

	// +optional
	Status RepoStatus `json:"status,omitempty"`
}

// RepoSpec contains spec fields for Repo.
// +protobuf=true
type RepoSpec struct {
	// Repo version string, corresponds to how policy importer should handle the
	// directory structure (implicit assumptions).
	Version string `json:"version" protobuf:"bytes,1,opt,name=version"`
}

// RepoStatus contains status fields for Repo.
// +protobuf=true
type RepoStatus struct {
	// +optional
	Source RepoSourceStatus `json:"source,omitempty"`

	// +optional
	Import RepoImportStatus `json:"import,omitempty"`

	// +optional
	Sync RepoSyncStatus `json:"sync,omitempty"`
}

// RepoSourceStatus contains status fields for the Repo's source of truth.
// +protobuf=true
type RepoSourceStatus struct {
	// Most recent version token seen in the source of truth (eg the repo). This token is updated as
	// soon as the policy importer sees a new change in the repo.
	// +optional
	Token string `json:"token,omitempty"`
}

// RepoImportStatus contains status fields for the import of the Repo.
// +protobuf=true
type RepoImportStatus struct {
	// Most recent version token imported from the source of truth into Nomos CRs. This token is
	// updated once the importer finishes processing a change, whether or not there were errors
	// during the import.
	// +optional
	Token string `json:"token,omitempty"`

	// LastUpdate is the timestamp of when this status was updated by the Importer.
	// +optional
	LastUpdate metav1.Time `json:"lastUpdate,omitempty"`

	// TODO(ekitson): add importer errors
}

// RepoSyncStatus contains status fields for the sync of the Repo.
// +protobuf=true
type RepoSyncStatus struct {
	// Most recent version token synced from Nomos CRs to Nomos-managed resources. This token is
	// updated  once the syncer finishes processing a change, whether or not there were errors during
	// the sync.
	// +optional
	Token string `json:"token,omitempty"`

	// LastUpdate is the timestamp of when this status was updated by the Importer.
	// +optional
	LastUpdate metav1.Time `json:"lastUpdate,omitempty"`

	// TODO(ekitson): add syncer errors
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RepoList holds a list of Repo resources.
// +protobuf=true
type RepoList struct {
	metav1.TypeMeta `json:",inline"`

	// Standard object's metadata.
	// +optional
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Items is a list of Repo declarations.
	Items []Repo `json:"items" protobuf:"bytes,2,rep,name=items"`
}

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// HierarchicalQuota holds hierarchical ResourceQuota information.
type HierarchicalQuota struct {
	metav1.TypeMeta `json:",inline"`

	// Standard object's metadata. The Name field of the policy node must match the namespace name.
	// +optional
	metav1.ObjectMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`

	// The actual object definition, per K8S object definition style.
	// +optional
	Spec HierarchicalQuotaSpec `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// HierarchicalQuotaList holds a list of HierarchicalQuota resources.
type HierarchicalQuotaList struct {
	metav1.TypeMeta `json:",inline"`

	// Standard object's metadata.
	// +optional
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Items is a list of HierarchicalQuotas.
	Items []HierarchicalQuota `json:"items" protobuf:"bytes,2,rep,name=items"`
}

// HierarchicalQuotaSpec holds fields for hierarchical quota definition.
type HierarchicalQuotaSpec struct {
	Hierarchy HierarchicalQuotaNode `json:"hierarchy"`
}

// HierarchicalQuotaNode is an element of a quota hierarchy.
type HierarchicalQuotaNode struct {
	// Name is the name of the namespace or abstract namespace
	// +optional
	Name string `json:"name,omitempty"`

	// Type is the type of the hierarchical quota node.
	Type HierarchyNodeType `json:"type,omitempty"`
	// +optional
	ResourceQuotaV1 *v1.ResourceQuota `json:"resourceQuotaV1,omitempty"`

	// Children are the child nodes of this node.  This will be populated for abstract namespaces.
	// +optional
	Children []HierarchicalQuotaNode `json:"children,omitempty"`
}

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// HierarchyConfig is used for configuring the HierarchyModeType for managed resources.
type HierarchyConfig struct {
	metav1.TypeMeta `json:",inline"`

	// Standard object's metadata. The Name field of the policy node must match the namespace name.
	// +optional
	metav1.ObjectMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`

	// Spec is the standard spec field.
	Spec HierarchyConfigSpec `json:"spec"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// HierarchyConfigList holds a list of HierarchyConfig resources.
type HierarchyConfigList struct {
	metav1.TypeMeta `json:",inline"`

	// Standard object's metadata.
	// +optional
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Items is a list of HierarchyConfigs.
	Items []HierarchyConfig `json:"items" protobuf:"bytes,2,rep,name=items"`
}

// HierarchyConfigSpec specifies the HierarchyConfigResources.
type HierarchyConfigSpec struct {
	Resources []HierarchyConfigResource `json:"resources" protobuf:"bytes,2,rep,name=resources"`
}

// HierarchyConfigResource specifies the HierarchyModeType based on the matching Groups and Kinds enabled.
type HierarchyConfigResource struct {
	// Group is the name of the APIGroup that contains the resources.
	// +optional
	Group string `json:"group,omitempty" protobuf:"bytes,1,rep,name=group"`
	// Kinds is a list of kinds this rule applies to.
	// +optional
	Kinds []string `json:"kinds,omitempty" protobuf:"bytes,2,rep,name=kinds"`
	// HierarchyMode specifies how the object is treated when it appears in an abstract namespace.
	// The default is off, meaning objects cannot appear in an abstract namespace. For RoleBinding,
	// the default is "inherit". For ResourceQuota, the default is "hierarchicalQuota".
	// +optional
	HierarchyMode HierarchyModeType `json:"hierarchyMode,omitempty" protobuf:"bytes,3,opt,name=hierarchyMode"`
}
