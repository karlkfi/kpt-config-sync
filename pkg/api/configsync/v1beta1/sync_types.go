package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SyncStatus provides a common type that is embedded in RepoSyncStatus and RootSyncStatus.
type SyncStatus struct {
	// ObservedGeneration is the most recent generation observed for the sync resource.
	// It corresponds to the it's generation, which is updated on mutation by the API Server.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Reconciler is the name of the reconciler process which corresponds to the
	// sync resource.
	// +optional
	Reconciler string `json:"reconciler,omitempty"`

	// LastSyncedCommit describes the most recent commit hash that is successfully synced.
	// +optional
	LastSyncedCommit string `json:"lastSyncedCommit,omitempty"`

	// Source contains fields describing the status of a *Sync's source of
	// truth.
	// +optional
	Source GitSourceStatus `json:"source,omitempty"`

	// Rendering contains fields describing the status of rendering resources from
	// the source of truth.
	// +optional
	Rendering RenderingStatus `json:"rendering,omitempty"`

	// Sync contains fields describing the status of syncing resources from the
	// source of truth to the cluster.
	// +optional
	Sync GitSyncStatus `json:"sync,omitempty"`
}

// GitSourceStatus describes the status of a git source-of-truth
type GitSourceStatus struct {
	// Git contains fields describing the status of a Git source of truth.
	// +optional
	Git GitStatus `json:"gitStatus,omitempty"`

	// Commit is the hash of the most recent commit seen in the source of truth.
	// +optional
	Commit string `json:"commit,omitempty"`

	// LastUpdate is the timestamp of when this status was last updated by a
	// reconciler.
	// +nullable
	// +optional
	LastUpdate metav1.Time `json:"lastUpdate,omitempty"`

	// Errors is a list of any errors that occurred while reading from the source of truth.
	// +optional
	Errors []ConfigSyncError `json:"errors,omitempty"`
}

// RenderingStatus describes the status of rendering the source DRY configs to the WET format.
type RenderingStatus struct {
	// Git contains fields describing the status of a Git source of truth.
	// +optional
	Git GitStatus `json:"gitStatus,omitempty"`

	// Commit is the hash of the commit in the source of truth that is rendered.
	// +optional
	Commit string `json:"commit,omitempty"`

	// LastUpdate is the timestamp of when this status was last updated by a
	// reconciler.
	// +nullable
	// +optional
	LastUpdate metav1.Time `json:"lastUpdate,omitempty"`

	// Human-readable message describes details about the rendering status.
	Message string `json:"message,omitempty"`

	// Errors is a list of any errors that occurred while rendering the source of truth.
	// +optional
	Errors []ConfigSyncError `json:"errors,omitempty"`
}

// GitSyncStatus provides the status of the syncing of resources from a git source-of-truth on to the cluster
type GitSyncStatus struct {
	// Git contains fields describing the status of a Git source of truth.
	// +optional
	Git GitStatus `json:"gitStatus,omitempty"`
	// Commit is the hash of the most recent commit that was synced to the
	// cluster. This value is updated even when a commit is only partially synced
	// due to an  error.
	// +optional
	Commit string `json:"commit,omitempty"`

	// LastUpdate is the timestamp of when this status was last updated by a
	// reconciler.
	// +nullable
	// +optional
	LastUpdate metav1.Time `json:"lastUpdate,omitempty"`

	// Errors is a list of any errors that occurred while applying the resources
	// from the change indicated by Commit.
	// +optional
	Errors []ConfigSyncError `json:"errors,omitempty"`
}

// GitStatus describes the status of a Git source of truth.
type GitStatus struct {
	// Repo is the git repository URL being synced from.
	Repo string `json:"repo"`

	// Revision is the git revision (tag, ref, or commit) being fetched.
	Revision string `json:"revision"`

	// Branch is the git branch being fetched
	Branch string `json:"branch"`

	// Dir is the path within the Git repository that represents the top level of the repo to sync.
	// Default: the root directory of the repository
	Dir string `json:"dir"`
}

// ConfigSyncError represents an error that occurs while parsing, applying, or
// remediating a resource.
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
