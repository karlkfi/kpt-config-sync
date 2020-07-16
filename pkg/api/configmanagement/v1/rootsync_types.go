package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// +kubebuilder:object:root=true

// RootSyncSpec defines the desired state of RootSync
type RootSyncSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster

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

// +kubebuilder:object:root=true

// RootSyncStatus defines the observed state of RootSync
type RootSyncStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true

// RootSync is the Schema for the rootsyncs API
type RootSync struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// +optional
	Spec RootSyncSpec `json:"spec,omitempty"`
	// +optional
	Status RootSyncStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// RootSyncList contains a list of RootSync
type RootSyncList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RootSync `json:"items"`
}
