package rootsync

import (
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Local alias to enable unit test mocking.
var now = metav1.Now

// ClearCondition sets the specified condition to False if it is currently
// defined as True. If the condition is unspecified, then it is left that way.
func ClearCondition(rs *v1alpha1.RootSync, condType v1alpha1.RootSyncConditionType) {
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

// IsReconciling returns true if the given RootSync has a True Reconciling condition.
func IsReconciling(rs *v1alpha1.RootSync) bool {
	cond := GetCondition(rs.Status.Conditions, v1alpha1.RootSyncReconciling)
	return cond != nil && cond.Status == metav1.ConditionTrue
}

// IsStalled returns true if the given RootSync has a True Stalled condition.
func IsStalled(rs *v1alpha1.RootSync) bool {
	cond := GetCondition(rs.Status.Conditions, v1alpha1.RootSyncStalled)
	return cond != nil && cond.Status == metav1.ConditionTrue
}

// ReconcilingMessage returns the message from a True Reconciling condition or
// an empty string if no True Reconciling condition was found.
func ReconcilingMessage(rs *v1alpha1.RootSync) string {
	cond := GetCondition(rs.Status.Conditions, v1alpha1.RootSyncReconciling)
	if cond == nil || cond.Status == metav1.ConditionFalse {
		return ""
	}
	return cond.Message
}

// StalledMessage returns the message from a True Stalled condition or an empty
// string if no True Stalled condition was found.
func StalledMessage(rs *v1alpha1.RootSync) string {
	cond := GetCondition(rs.Status.Conditions, v1alpha1.RootSyncStalled)
	if cond == nil || cond.Status == metav1.ConditionFalse {
		return ""
	}
	return cond.Message
}

// SetReconciling sets the Reconciling condition to True.
func SetReconciling(rs *v1alpha1.RootSync, reason, message string) {
	if setCondition(rs, v1alpha1.RootSyncReconciling, metav1.ConditionTrue, reason, message, "", nil) {
		// Only remove the Syncing condition when the Reconciling condition status is updated from false to true.
		removeCondition(rs, v1alpha1.RootSyncSyncing)
	}
}

// SetStalled sets the Stalled condition to True.
func SetStalled(rs *v1alpha1.RootSync, reason string, err error) {
	if setCondition(rs, v1alpha1.RootSyncStalled, metav1.ConditionTrue, reason, err.Error(), "", nil) {
		// Only remove the Syncing condition when the Stalled condition status is updated from false to true.
		removeCondition(rs, v1alpha1.RootSyncSyncing)
	}
}

// SetSyncing sets the Syncing condition.
func SetSyncing(rs *v1alpha1.RootSync, status bool, reason, message, commit string, errs []v1alpha1.ConfigSyncError) {
	var conditionStatus metav1.ConditionStatus
	if status {
		conditionStatus = metav1.ConditionTrue
	} else {
		conditionStatus = metav1.ConditionFalse
	}
	setCondition(rs, v1alpha1.RootSyncSyncing, conditionStatus, reason, message, commit, errs)
}

// setCondition adds or updates the specified condition.
// It returns a boolean indicating if the condition status is transited.
func setCondition(rs *v1alpha1.RootSync, condType v1alpha1.RootSyncConditionType, status metav1.ConditionStatus, reason, message, commit string, errs []v1alpha1.ConfigSyncError) bool {
	conditionTransited := false
	condition := GetCondition(rs.Status.Conditions, condType)
	if condition == nil {
		i := len(rs.Status.Conditions)
		rs.Status.Conditions = append(rs.Status.Conditions, v1alpha1.RootSyncCondition{Type: condType})
		condition = &rs.Status.Conditions[i]
	}

	time := now()
	if condition.Status != status {
		condition.Status = status
		condition.LastTransitionTime = time
		conditionTransited = true
	}
	condition.Reason = reason
	condition.Message = message
	condition.Commit = commit
	condition.Errors = errs
	condition.LastUpdateTime = time
	return conditionTransited
}

// GetCondition returns the condition with the provided type.
func GetCondition(conditions []v1alpha1.RootSyncCondition, condType v1alpha1.RootSyncConditionType) *v1alpha1.RootSyncCondition {
	for i, condition := range conditions {
		if condition.Type == condType {
			return &conditions[i]
		}
	}
	return nil
}

// removeCondition removes the RootSync condition with the provided type.
func removeCondition(rs *v1alpha1.RootSync, condType v1alpha1.RootSyncConditionType) {
	rs.Status.Conditions = filterOutCondition(rs.Status.Conditions, condType)
}

// filterOutCondition returns a new slice of RootSync conditions without conditions with the provided type.
func filterOutCondition(conditions []v1alpha1.RootSyncCondition, condType v1alpha1.RootSyncConditionType) []v1alpha1.RootSyncCondition {
	var newConditions []v1alpha1.RootSyncCondition
	for _, c := range conditions {
		if c.Type == condType {
			continue
		}
		newConditions = append(newConditions, c)
	}
	return newConditions
}
