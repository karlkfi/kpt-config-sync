package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true
// +protobuf=true

// RepoSync is the Schema for the reposyncs API
type RepoSync struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// +optional
	Spec RepoSyncSpec `json:"spec,omitempty"`
	// +optional
	Status RepoSyncsStatus `json:"status,omitempty"`
}

// +kubebuilder:object:generate=true
// +protobuf=true

// RepoSyncSpec defines the desired state of a RepoSync.
type RepoSyncSpec struct {
	// SourceFormat specifies how the repository is formatted.
	// See documentation for specifics of what these options do.
	//
	// Must be one of hierarchy, unstructured. Optional. Set to
	// hierarchy if not specified.
	//
	// The validation of this is case-sensitive.
	// +kubebuilder:validation:Pattern=^(hierarchy|unstructured|)$
	// +optional
	SourceFormat string `json:"sourceFormat,omitempty"`

	// Git contains configuration specific to importing policies from a Git repo.
	// +optional
	Git `json:"git,omitempty"`
}

// +kubebuilder:object:generate=true
// +protobuf=true

// RepoSyncsStatus defines the observed state of a RepoSync.
// Note that the extra s is required to deconflict with the pre-existing
// RepoSyncStatus type.
type RepoSyncsStatus struct {
	// ObservedGeneration is the most recent generation observed for the RepoSync.
	// It corresponds to the RepoSync's generation, which is updated on mutation
	// by the API Server.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Reconciler is the name of the reconciler process which corresponds to the
	// RepoSync.
	// +optional
	Reconciler string `json:"reconciler,omitempty"`

	// Conditions represents the latest available observations of the RepoSync's
	// current state.
	// +optional
	Conditions []RepoSyncCondition `json:"conditions,omitempty"`

	// Source contains fields describing the status of the RepoSync's source of
	// truth.
	// +optional
	Source RepoSyncSourceStatus `json:"source,omitempty"`

	// Sync contains fields describing the status of syncing resources from the
	// source of truth to the cluster.
	// +optional
	Sync RepoSyncSyncStatus `json:"sync,omitempty"`
}

// RepoSyncConditionType is an enum of types of conditions for RepoSyncs.
type RepoSyncConditionType string

// These are valid conditions of a RepoSync.
const (
	// The following conditions are currently recommended as "standard" resource
	// conditions which are supported by kstatus and kpt:
	// https://github.com/kubernetes-sigs/cli-utils/tree/master/pkg/kstatus#conditions

	// RepoSyncReconciling means that the RepoSync's spec has not yet been fully
	// reconciled/handled by the RepoSync controller.
	RepoSyncReconciling RepoSyncConditionType = "Reconciling"
	// RepoSyncStalled means that the RepoSync controller has not been able to
	// make progress towards reconciling the RepoSync.
	RepoSyncStalled RepoSyncConditionType = "Stalled"
)

// +kubebuilder:object:generate=true
// +protobuf=true

// RepoSyncCondition describes the state of a RepoSync at a certain point.
type RepoSyncCondition struct {
	// Type of RepoSync condition.
	Type RepoSyncConditionType `json:"type"`
	// Status of the condition, one of True, False, Unknown.
	Status metav1.ConditionStatus `json:"status"`
	// The last time this condition was updated.
	// +optional
	LastUpdateTime metav1.Time `json:"lastUpdateTime,omitempty"`
	// Last time the condition transitioned from one status to another.
	// +optional
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
	// The reason for the condition's last transition.
	// +optional
	Reason string `json:"reason,omitempty"`
	// A human readable message indicating details about the transition.
	// +optional
	Message string `json:"message,omitempty"`
}

// +kubebuilder:object:generate=true
// +protobuf=true

// RepoSyncSourceStatus describes the status of the RepoSync's source of truth.
type RepoSyncSourceStatus struct {
	// Git contains fields describing the status of a Git source of truth.
	// +optional
	Git GitStatus `json:"gitStatus,omitempty"`

	// Commit is the hash of the most recent commit seen in the source of truth.
	// +optional
	Commit string `json:"commit,omitempty"`

	// Errors is a list of any errors that occurred while reading from the source of truth.
	// +optional
	Errors []ConfigSyncError `json:"errors,omitempty"`
}

// +protobuf=true

// GitStatus describes the status of a Git source of truth.
type GitStatus struct {
	// Repo is the git repository URL being synced from.
	Repo string `json:"repo"`

	// Revision is the git revision (tag, branch, ref or commit) being fetched.
	Revision string `json:"revision"`

	// Dir is the absolute path of the directory that contains the local policy.
	Dir string `json:"dir"`
}

// +kubebuilder:object:generate=true
// +protobuf=true

// ConfigSyncError represents an error that occurs while parsing, applying, or
// remediating a resource. We can't re-use the existing ConfigManagementError
// type because it relies on schema.GroupVersionKind which does not have JSON
// encoding annotations.
type ConfigSyncError struct {
	// Code is the error code of this particular error.  Error codes are numeric strings,
	// like "1012".
	Code string `json:"code"`

	// ErrorMessage describes the error that occurred.
	ErrorMessage string `json:"errorMessage"`

	// Resources describes the resources associated with this error, if any.
	// +optional
	Resources []ResourceRef `json:"errorResources,omitempty"`
}

// +protobuf=true

// ResourceRef contains the identification bits of a single managed resource.
type ResourceRef struct {
	// SourcePath is the repo-relative slash path to where the config is defined.
	// This field may be empty for errors that are not associated with a specific
	// config file.
	// +optional
	SourcePath string `json:"sourcePath,omitempty"`

	// Name is the name of the affected K8S resource. This field may be empty for
	// errors that are not associated with a specific resource.
	// +optional
	Name string `json:"name,omitempty"`

	// Namespace is the namespace of the affected K8S resource. This field may be
	// empty for errors that are associated with a cluster-scoped resource or not
	// associated with a specific resource.
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// GVK is the GroupVersionKind of the affected K8S resource. This field may be
	// empty for errors that are not associated with a specific resource.
	// +optional
	GVK metav1.GroupVersionKind `json:"gvk,omitempty"`
}

// +kubebuilder:object:generate=true
// +protobuf=true

// RepoSyncSyncStatus describes the status of syncing resources from the source
// of truth to the cluster.
type RepoSyncSyncStatus struct {
	// Commit is the hash of the most recent commit that was synced to the
	// cluster. This value is updated even when a commit is only partially synced
	// due to an  error.
	// +optional
	Commit string `json:"commit,omitempty"`

	// LastUpdate is the timestamp of when this status was last updated by a
	// reconciler.
	// +optional
	LastUpdate metav1.Time `json:"lastUpdate,omitempty"`

	// Errors is a list of any errors that occurred while applying the resources
	// from the change indicated by Commit.
	// +optional
	Errors []ConfigSyncError `json:"errors,omitempty"`
}

// +kubebuilder:object:root=true

// RepoSyncList contains a list of RepoSync
type RepoSyncList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RepoSync `json:"items"`
}
