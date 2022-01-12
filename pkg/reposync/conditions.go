package reposync

import (
	"github.com/google/nomos/pkg/api/configsync/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Local alias to enable unit test mocking.
var now = metav1.Now

// IsReconciling returns true if the given RepoSync has a True Reconciling condition.
func IsReconciling(rs *v1beta1.RepoSync) bool {
	cond := GetCondition(rs.Status.Conditions, v1beta1.RepoSyncReconciling)
	return cond != nil && cond.Status == metav1.ConditionTrue
}

// IsStalled returns true if the given RepoSync has a True Stalled condition.
func IsStalled(rs *v1beta1.RepoSync) bool {
	cond := GetCondition(rs.Status.Conditions, v1beta1.RepoSyncStalled)
	return cond != nil && cond.Status == metav1.ConditionTrue
}

// ReconcilingMessage returns the message from a True Reconciling condition or
// an empty string if no True Reconciling condition was found.
func ReconcilingMessage(rs *v1beta1.RepoSync) string {
	cond := GetCondition(rs.Status.Conditions, v1beta1.RepoSyncReconciling)
	if cond == nil || cond.Status == metav1.ConditionFalse {
		return ""
	}
	return cond.Message
}

// StalledMessage returns the message from a True Stalled condition or an empty
// string if no True Stalled condition was found.
func StalledMessage(rs *v1beta1.RepoSync) string {
	cond := GetCondition(rs.Status.Conditions, v1beta1.RepoSyncStalled)
	if cond == nil || cond.Status == metav1.ConditionFalse {
		return ""
	}
	return cond.Message
}

// ClearCondition sets the specified condition to False if it is currently
// defined as True. If the condition is unspecified, then it is left that way.
func ClearCondition(rs *v1beta1.RepoSync, condType v1beta1.RepoSyncConditionType) {
	condition := GetCondition(rs.Status.Conditions, condType)
	if condition == nil {
		return
	}

	if condition.Status == metav1.ConditionFalse {
		return
	}

	time := now()
	condition.Status = metav1.ConditionFalse
	condition.Reason = ""
	condition.Message = ""
	condition.LastTransitionTime = time
	condition.LastUpdateTime = time
}

// SetReconciling sets the Reconciling condition to True.
func SetReconciling(rs *v1beta1.RepoSync, reason, message string) {
	if setCondition(rs, v1beta1.RepoSyncReconciling, metav1.ConditionTrue, reason, message, "", nil, now()) {
		removeCondition(rs, v1beta1.RepoSyncSyncing)
	}
}

// SetStalled sets the Stalled condition to True.
func SetStalled(rs *v1beta1.RepoSync, reason string, err error) {
	if setCondition(rs, v1beta1.RepoSyncStalled, metav1.ConditionTrue, reason, err.Error(), "", nil, now()) {
		removeCondition(rs, v1beta1.RepoSyncSyncing)
	}
}

// SetSyncing sets the Syncing condition.
func SetSyncing(rs *v1beta1.RepoSync, status bool, reason, message, commit string, errs []v1beta1.ConfigSyncError, lastUpdate metav1.Time) {
	var conditionStatus metav1.ConditionStatus
	if status {
		conditionStatus = metav1.ConditionTrue
	} else {
		conditionStatus = metav1.ConditionFalse
	}
	setCondition(rs, v1beta1.RepoSyncSyncing, conditionStatus, reason, message, commit, errs, lastUpdate)
}

// setCondition adds or updates the specified condition with a True status.
// It returns a boolean indicating if the condition status is transited.
func setCondition(rs *v1beta1.RepoSync, condType v1beta1.RepoSyncConditionType, status metav1.ConditionStatus, reason, message, commit string, errs []v1beta1.ConfigSyncError, lastUpdate metav1.Time) bool {
	conditionTransited := false
	condition := GetCondition(rs.Status.Conditions, condType)
	if condition == nil {
		i := len(rs.Status.Conditions)
		rs.Status.Conditions = append(rs.Status.Conditions, v1beta1.RepoSyncCondition{Type: condType})
		condition = &rs.Status.Conditions[i]
	}

	if condition.Status != status {
		condition.Status = status
		condition.LastTransitionTime = lastUpdate
		conditionTransited = true
	}
	condition.Reason = reason
	condition.Message = message
	condition.Commit = commit
	condition.Errors = errs
	condition.LastUpdateTime = lastUpdate

	return conditionTransited
}

// GetCondition returns the condition with the provided type.
func GetCondition(conditions []v1beta1.RepoSyncCondition, condType v1beta1.RepoSyncConditionType) *v1beta1.RepoSyncCondition {
	for i, condition := range conditions {
		if condition.Type == condType {
			return &conditions[i]
		}
	}
	return nil
}

// removeCondition removes the RepoSync condition with the provided type.
func removeCondition(rs *v1beta1.RepoSync, condType v1beta1.RepoSyncConditionType) {
	rs.Status.Conditions = filterOutCondition(rs.Status.Conditions, condType)
}

// filterOutCondition returns a new slice of RepoSync conditions without conditions with the provided type.
func filterOutCondition(conditions []v1beta1.RepoSyncCondition, condType v1beta1.RepoSyncConditionType) []v1beta1.RepoSyncCondition {
	var newConditions []v1beta1.RepoSyncCondition
	for _, c := range conditions {
		if c.Type == condType {
			continue
		}
		newConditions = append(newConditions, c)
	}
	return newConditions
}
