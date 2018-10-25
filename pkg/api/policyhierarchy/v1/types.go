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
	v1alpha1 "github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
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
	ClusterRolesV1 []rbacv1.ClusterRole `json:"clusterRolesV1,omitempty" protobuf:"bytes,1,rep,name=clusterRolesV1"`

	// +optional
	ClusterRoleBindingsV1 []rbacv1.ClusterRoleBinding `json:"clusterRoleBindingsV1,omitempty" protobuf:"bytes,2,rep,name=clusterRoleBindingsV1"`

	// +optional
	PodSecurityPoliciesV1Beta1 []extensionsv1beta1.PodSecurityPolicy `json:"podSecurityPolicyV1Beta1,omitempty" protobuf:"bytes,3,rep,name=podSecurityPolicyV1Beta1"`

	// ImportToken indicates the version of the ClusterPolicy last imported from the source of truth.
	// +optional
	ImportToken string `json:"importToken,omitempty" protobuf:"bytes,4,opt,name=importToken"`

	// ImportTime is the timestamp of when the ClusterPolicy was updated by the Importer.
	// +optional
	ImportTime metav1.Time `json:"importTime,omitempty" protobuf:"bytes,5,opt,name=importTime"`

	// Resources contains namespace scoped resources that are synced to the API server.
	// +optional
	Resources []GenericResources `json:"resources,omitempty" protobuf:"bytes,8,opt,name=resources"`
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
	RolesV1 []rbacv1.Role `json:"rolesV1,omitempty" protobuf:"bytes,3,rep,name=rolesV1"`

	// +optional
	RoleBindingsV1 []rbacv1.RoleBinding `json:"roleBindingsV1,omitempty" protobuf:"bytes,4,rep,name=roleBindingsV1"`

	// +optional
	ResourceQuotaV1 *corev1.ResourceQuota `json:"resourceQuotaV1,omitempty" protobuf:"bytes,5,opt,name=resourceQuotaV1"`

	// ImportToken indicates the version of the PolicyNode last imported from the source of truth.
	// +optional
	ImportToken string `json:"importToken,omitempty" protobuf:"bytes,6,opt,name=importToken"`

	// ImportTime is the timestamp of when the PolicyNode was updated by the Importer.
	// +optional
	ImportTime metav1.Time `json:"importTime,omitempty" protobuf:"bytes,7,opt,name=importTime"`

	// Resources contains namespace scoped resources that are synced to the API server.
	// +optional
	Resources []GenericResources `json:"resources,omitempty" protobuf:"bytes,8,opt,name=resources"`
}

// PolicyNodeStatus contains fields that define the status of a PolicyNode. The fields related to Syncer
// will never be populated for PolicySpaces since they are flattened down to child Namespaces.
// +protobuf=true
type PolicyNodeStatus struct {
	// SyncToken indicates the version of that policy that the Syncer last attempted to update from.
	// +optional
	SyncToken string `json:"syncToken,omitempty" protobuf:"bytes,1,rep,name=syncToken"`

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

// AllPolicies holds things that Importer wants to sync. It is only used in-process, not written
// directly as a Kubernetes resource.
type AllPolicies struct {
	// Map of names to PolicyNodes.
	// +optional
	PolicyNodes map[string]PolicyNode `protobuf:"bytes,1,rep,name=policyNodes"`
	// +optional
	ClusterPolicy *ClusterPolicy `protobuf:"bytes,2,opt,name=clusterPolicy"`
	// Map of names to Syncs.
	// +optional
	Syncs map[string]v1alpha1.Sync `protobuf:"bytes,3,rep,name=syncs"`
}
