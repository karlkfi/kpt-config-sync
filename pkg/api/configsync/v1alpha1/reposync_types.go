package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="RenderingCommit",type="string",JSONPath=".status.rendering.commit"
// +kubebuilder:printcolumn:name="RenderingErrorCount",type="integer",JSONPath=".status.rendering.errorSummary.totalCount"
// +kubebuilder:printcolumn:name="SourceCommit",type="string",JSONPath=".status.source.commit"
// +kubebuilder:printcolumn:name="SourceErrorCount",type="integer",JSONPath=".status.source.errorSummary.totalCount"
// +kubebuilder:printcolumn:name="SyncCommit",type="string",JSONPath=".status.sync.commit"
// +kubebuilder:printcolumn:name="SyncErrorCount",type="integer",JSONPath=".status.sync.errorSummary.totalCount"

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
	// RepoSyncSyncing means that the namespace reconciler is processing a commit.
	RepoSyncSyncing RepoSyncConditionType = "Syncing"
)

// RepoSyncCondition describes the state of a RepoSync at a certain point.
type RepoSyncCondition struct {
	// Type of RepoSync condition.
	Type RepoSyncConditionType `json:"type"`
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
	// Commit is the hash of the commit in the source of truth.
	// +optional
	Commit string `json:"commit,omitempty"`
	// Errors is a list of errors that occurred in the process.
	// This field is used to track errors when the condition type is Reconciling or Stalled.
	// When the condition type is Syncing, the `errorSourceRefs` field is used instead to
	// avoid duplicating errors between `status.conditions` and `status.rendering|source|sync`.
	// +optional
	Errors []ConfigSyncError `json:"errors,omitempty"`
	// errorSourceRefs track the origination(s) of errors when the condition type is Syncing.
	// +optional
	ErrorSourceRefs []ErrorSource `json:"errorSourceRefs,omitempty"`
	// errorSummary summarizes the errors in the `errors` field when the condition type is Reconciling or Stalled,
	// and summarizes the errors referred in the `errorsSourceRefs` field when the condition type is Syncing.
	// +optional
	ErrorSummary *ErrorSummary `json:"errorSummary,omitempty"`
}

// +kubebuilder:object:root=true

// RepoSyncList contains a list of RepoSync
type RepoSyncList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RepoSync `json:"items"`
}
