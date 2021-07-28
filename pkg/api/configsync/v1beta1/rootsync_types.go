package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="RenderingCommit",type="string",JSONPath=".status.rendering.commit"
// +kubebuilder:printcolumn:name="SourceCommit",type="string",JSONPath=".status.source.commit"
// +kubebuilder:printcolumn:name="SyncCommit",type="string",JSONPath=".status.sync.commit"
// +kubebuilder:storageversion

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

// RootSyncSpec defines the desired state of RootSync
type RootSyncSpec struct {
	// sourceFormat specifies how the repository is formatted.
	// See documentation for specifics of what these options do.
	//
	// Must be one of hierarchy, unstructured. Optional. Set to
	// hierarchy if not specified.
	//
	// The validation of this is case-sensitive.
	// +kubebuilder:validation:Pattern=^(hierarchy|unstructured|)$
	// +optional
	SourceFormat string `json:"sourceFormat,omitempty"`

	// git contains configuration specific to importing policies from a Git repo.
	// +optional
	Git `json:"git,omitempty"`

	// override allows to override the settings for a root reconciler.
	// +nullable
	// +optional
	Override OverrideSpec `json:"override,omitempty"`
}

// RootSyncStatus defines the observed state of RootSync
type RootSyncStatus struct {
	SyncStatus `json:",inline"`

	// Conditions represents the latest available observations of the RootSync's
	// current state.
	// +optional
	Conditions []RootSyncCondition `json:"conditions,omitempty"`
}

// RootSyncConditionType is an enum of types of conditions for RootSyncs.
type RootSyncConditionType string

// These are valid conditions of a RootSync.
const (
	// The following conditions are currently recommended as "standard" resource
	// conditions which are supported by kstatus and kpt:
	// https://github.com/kubernetes-sigs/cli-utils/tree/master/pkg/kstatus#conditions

	// RootSyncReconciling means that the RootSync's spec has not yet been fully
	// reconciled/handled by the RootSync controller.
	RootSyncReconciling RootSyncConditionType = "Reconciling"
	// RootSyncStalled means that the RootSync controller has not been able to
	// make progress towards reconciling the RootSync.
	RootSyncStalled RootSyncConditionType = "Stalled"
)

// RootSyncCondition describes the state of a RootSync at a certain point.
type RootSyncCondition struct {
	// Type of RootSync condition.
	Type RootSyncConditionType `json:"type"`
	// Status of the condition, one of True, False, Unknown.
	Status metav1.ConditionStatus `json:"status"`
	// The last time this condition was updated.
	// +nullable
	// +optional
	LastUpdateTime metav1.Time `json:"lastUpdateTime,omitempty"`
	// Last time the condition transitioned from one status to another.
	// +nullable
	// +optional
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
	// The reason for the condition's last transition.
	// +optional
	Reason string `json:"reason,omitempty"`
	// A human readable message indicating details about the transition.
	// +optional
	Message string `json:"message,omitempty"`
}

// +kubebuilder:object:root=true

// RootSyncList contains a list of RootSync
type RootSyncList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RootSync `json:"items"`
}
