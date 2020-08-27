package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:generate=true
// +protobuf=true

// MultiRepoSyncSpec provides a common type that is embedded in RepoSyncSpec and RootSyncSpec
type MultiRepoSyncSpec struct {
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

// MultiRepoSyncStatus provides a common type that is embedded in RepoSyncsStatus and RootSyncStatus
type MultiRepoSyncStatus struct {
	// ObservedGeneration is the most recent generation observed for the RepoSync.
	// It corresponds to the RepoSync's generation, which is updated on mutation
	// by the API Server.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Reconciler is the name of the reconciler process which corresponds to the
	// RepoSync or RootSync.
	// +optional
	Reconciler string `json:"reconciler,omitempty"`
}

// +kubebuilder:object:generate=true
// +protobuf=true

// MultiRepoSyncSourceStatus provides a common type that is embedded in RepoSyncSourceStatus and RootSyncSourceStatus
type MultiRepoSyncSourceStatus struct {
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

// +kubebuilder:object:generate=true
// +protobuf=true

// MultiRepoSyncSyncStatus provides a common type that is embedded in RepoSyncSyncStatus and RootSyncSyncStatus
type MultiRepoSyncSyncStatus struct {
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

// +kubebuilder:object:generate=true
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

// +kubebuilder:object:generate=true
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
