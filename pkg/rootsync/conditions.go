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
	condition, ok := findCondition(rs.Status.Conditions, condType)
	if !ok {
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
func SetReconciling(rs *v1alpha1.RootSync, reason, message string) {
	setCondition(rs, v1alpha1.RootSyncReconciling, reason, message)
}

// SetStalled sets the Stalled condition to True.
func SetStalled(rs *v1alpha1.RootSync, reason string, err error) {
	setCondition(rs, v1alpha1.RootSyncStalled, reason, err.Error())
}

// setCondition adds or updates the specified condition with a True status.
func setCondition(rs *v1alpha1.RootSync, condType v1alpha1.RootSyncConditionType, reason, message string) {
	condition, ok := findCondition(rs.Status.Conditions, condType)
	if !ok {
		i := len(rs.Status.Conditions)
		rs.Status.Conditions = append(rs.Status.Conditions, v1alpha1.RootSyncCondition{Type: condType})
		condition = &rs.Status.Conditions[i]
	}

	time := now()
	if condition.Status != metav1.ConditionTrue {
		condition.Status = metav1.ConditionTrue
		condition.LastTransitionTime = time
	}
	condition.Reason = reason
	condition.Message = message
	condition.LastUpdateTime = time
}

func findCondition(conditions []v1alpha1.RootSyncCondition, condType v1alpha1.RootSyncConditionType) (*v1alpha1.RootSyncCondition, bool) {
	for i, condition := range conditions {
		if condition.Type == condType {
			return &conditions[i], true
		}
	}
	return nil, false
}
