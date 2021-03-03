package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="SourceCommit",type="string",JSONPath=".status.source.commit"
// +kubebuilder:printcolumn:name="SyncCommit",type="string",JSONPath=".status.sync.commit"
// +kubebuilder:storageversion

// RepoSync is the Schema for the reposyncs API
type RepoSync struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// +optional
	Spec RepoSyncSpec `json:"spec,omitempty"`
	// +optional
	Status RepoSyncStatus `json:"status,omitempty"`
}

// RepoSyncSpec defines the desired state of a RepoSync.
type RepoSyncSpec struct {
	SyncSpec `json:",inline"`
}

// RepoSyncStatus defines the observed state of a RepoSync.
type RepoSyncStatus struct {
	SyncStatus `json:",inline"`

	// Conditions represents the latest available observations of the RepoSync's
	// current state.
	// +optional
	Conditions []RepoSyncCondition `json:"conditions,omitempty"`
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

// +kubebuilder:object:root=true

// RepoSyncList contains a list of RepoSync
type RepoSyncList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RepoSync `json:"items"`
}
