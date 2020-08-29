package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +protobuf=true

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

// +kubebuilder:object:generate=true
// +protobuf=true

// RepoSyncSpec defines the desired state of a RepoSync.
type RepoSyncSpec struct {
	SyncSpec
}

// +kubebuilder:object:generate=true
// +protobuf=true

// RepoSyncStatus defines the observed state of a RepoSync.
type RepoSyncStatus struct {
	SyncStatus

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
	SyncSourceStatus
}

// +kubebuilder:object:generate=true
// +protobuf=true

// RepoSyncSyncStatus describes the status of syncing resources from the source
// of truth to the cluster.
type RepoSyncSyncStatus struct {
	SyncSyncStatus
}

// +kubebuilder:object:root=true

// RepoSyncList contains a list of RepoSync
type RepoSyncList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RepoSync `json:"items"`
}
