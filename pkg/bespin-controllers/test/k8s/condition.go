// nolint
package k8s

import (
	"time"

	bespinv1 "github.com/google/nomos/pkg/api/policyascode/v1"
	"github.com/google/nomos/pkg/bespin-controllers/test/apis/k8s/v1alpha1"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// nolint
func NewCustomReadyCondition(status v1.ConditionStatus, msg, rs string) bespinv1.Condition {
	return bespinv1.Condition{
		LastTransitionTime: metav1.Now().Format(time.RFC3339),
		Type:               v1alpha1.ReadyConditionType,
		Status:             status,
		Reason:             rs,
		Message:            msg,
	}
}

// nolint
func NewReadyCondition() bespinv1.Condition {
	return bespinv1.Condition{
		LastTransitionTime: metav1.Now().Format(time.RFC3339),
		Type:               v1alpha1.ReadyConditionType,
	}
}

// nolint
func NewReadyConditionWithError(err error) bespinv1.Condition {
	readyCondition := NewReadyCondition()
	readyCondition.Status = v1.ConditionFalse
	readyCondition.Message = err.Error()
	return readyCondition
}

// nolint
func ConditionsEqualIgnoreTransitionTime(c1, c2 bespinv1.Condition) bool {
	return c1.Message == c2.Message &&
		c1.Reason == c2.Reason &&
		c1.Status == c2.Status &&
		c1.Type == c2.Type
}

// nolint
func ConditionSlicesEqual(conditions1, conditions2 []bespinv1.Condition) bool {
	if len(conditions1) == 0 {
		return len(conditions2) == 0
	}
	for i, c1 := range conditions1 {
		c2 := conditions2[i]
		if !ConditionsEqualIgnoreTransitionTime(c1, c2) {
			return false
		}
	}
	return true
}
