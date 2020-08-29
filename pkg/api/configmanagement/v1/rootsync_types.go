package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

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

// +kubebuilder:object:generate=true

// RootSyncSpec defines the desired state of RootSync
type RootSyncSpec struct {
	MultiRepoSyncSpec `json:",inline"`
}

// +kubebuilder:object:generate=true

// RootSyncStatus defines the observed state of RootSync
type RootSyncStatus struct {
	MultiRepoSyncStatus `json:",inline"`

	// Conditions represents the latest available observations of the RootSync's
	// current state.
	// +optional
	Conditions []RootSyncCondition `json:"conditions,omitempty"`

	// Source contains fields describing the status of the RootSync's source of
	// truth.
	// +optional
	Source RootSyncSourceStatus `json:"source,omitempty"`

	// Sync contains fields describing the status of syncing resources from the
	// source of truth to the cluster.
	// +optional
	Sync RootSyncSyncStatus `json:"sync,omitempty"`
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

// +kubebuilder:object:generate=true
// +protobuf=true

// RootSyncCondition describes the state of a RootSync at a certain point.
type RootSyncCondition struct {
	// Type of RootSync condition.
	Type RootSyncConditionType `json:"type"`
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

// RootSyncSourceStatus describes the status of the RootSync's source of truth.
type RootSyncSourceStatus struct {
	MultiRepoSyncSourceStatus `json:",inline"`
}

// +kubebuilder:object:generate=true
// +protobuf=true

// RootSyncSyncStatus describes the status of syncing resources from the source
// of truth to the cluster.
type RootSyncSyncStatus struct {
	MultiRepoSyncSyncStatus `json:",inline"`
}

// +kubebuilder:object:root=true

// RootSyncList contains a list of RootSync
type RootSyncList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RootSync `json:"items"`
}
