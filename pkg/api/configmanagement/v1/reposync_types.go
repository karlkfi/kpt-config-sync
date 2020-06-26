package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// +kubebuilder:object:root=true

// RepoSyncSpec defines the desired state of RepoSync
type RepoSyncSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster

	// Git contains configuration specific to importing policies from a Git repo.
	// +optional
	Git `json:"git,omitempty"`
}

// +kubebuilder:object:root=true

// ReposSyncStatus defines the observed state of RepoSync
type ReposSyncStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true

// RepoSync is the Schema for the reposyncs API
type RepoSync struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// +optional
	Spec RepoSyncSpec `json:"spec,omitempty"`
	// +optional
	Status ReposSyncStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// RepoSyncList contains a list of RepoSync
type RepoSyncList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RepoSync `json:"items"`
}
