package rootsync

import (
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/testing/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const fakeConditionMessage = "Testing"

var testNow = metav1.Date(1, time.February, 3, 4, 5, 6, 7, time.Local)

func withConditions(conds ...v1alpha1.RootSyncCondition) core.MetaMutator {
	return func(o core.Object) {
		rs := o.(*v1alpha1.RootSync)
		rs.Status.Conditions = append(rs.Status.Conditions, conds...)
	}
}

func fakeCondition(condType v1alpha1.RootSyncConditionType, status metav1.ConditionStatus, strs ...string) v1alpha1.RootSyncCondition {
	rsc := v1alpha1.RootSyncCondition{
		Type:               condType,
		Status:             status,
		Reason:             "Test",
		Message:            fakeConditionMessage,
		LastUpdateTime:     testNow,
		LastTransitionTime: testNow,
	}
	if len(strs) > 0 {
		rsc.Reason = strs[0]
	}
	if len(strs) > 1 {
		rsc.Message = strs[1]
	}
	return rsc
}

func TestIsReconciling(t *testing.T) {
	testCases := []struct {
		name string
		rs   *v1alpha1.RootSync
		want bool
	}{
		{
			"Missing condition is false",
			fake.RootSyncObject(),
			false,
		},
		{
			"False condition is false",
			fake.RootSyncObject(withConditions(fakeCondition(v1alpha1.RootSyncReconciling, metav1.ConditionFalse))),
			false,
		},
		{
			"True condition is true",
			fake.RootSyncObject(withConditions(fakeCondition(v1alpha1.RootSyncReconciling, metav1.ConditionTrue), fakeCondition(v1alpha1.RootSyncStalled, metav1.ConditionFalse))),
			true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := IsReconciling(tc.rs)
			if got != tc.want {
				t.Errorf("got IsReconciling() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestIsStalled(t *testing.T) {
	testCases := []struct {
		name string
		rs   *v1alpha1.RootSync
		want bool
	}{
		{
			"Missing condition is false",
			fake.RootSyncObject(),
			false,
		},
		{
			"False condition is false",
			fake.RootSyncObject(withConditions(fakeCondition(v1alpha1.RootSyncStalled, metav1.ConditionFalse))),
			false,
		},
		{
			"True condition is true",
			fake.RootSyncObject(withConditions(fakeCondition(v1alpha1.RootSyncReconciling, metav1.ConditionFalse), fakeCondition(v1alpha1.RootSyncStalled, metav1.ConditionTrue))),
			true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := IsStalled(tc.rs)
			if got != tc.want {
				t.Errorf("got IsStalled() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestReconcilingMessage(t *testing.T) {
	testCases := []struct {
		name string
		rs   *v1alpha1.RootSync
		want string
	}{
		{
			"Missing condition is empty",
			fake.RootSyncObject(),
			"",
		},
		{
			"False condition is empty",
			fake.RootSyncObject(withConditions(fakeCondition(v1alpha1.RootSyncReconciling, metav1.ConditionFalse))),
			"",
		},
		{
			"True condition is its message",
			fake.RootSyncObject(withConditions(fakeCondition(v1alpha1.RootSyncReconciling, metav1.ConditionTrue), fakeCondition(v1alpha1.RootSyncStalled, metav1.ConditionFalse))),
			fakeConditionMessage,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := ReconcilingMessage(tc.rs)
			if got != tc.want {
				t.Errorf("got ReconcilingMessage() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestStalledMessage(t *testing.T) {
	testCases := []struct {
		name string
		rs   *v1alpha1.RootSync
		want string
	}{
		{
			"Missing condition is empty",
			fake.RootSyncObject(),
			"",
		},
		{
			"False condition is empty",
			fake.RootSyncObject(withConditions(fakeCondition(v1alpha1.RootSyncStalled, metav1.ConditionFalse))),
			"",
		},
		{
			"True condition is its message",
			fake.RootSyncObject(withConditions(fakeCondition(v1alpha1.RootSyncReconciling, metav1.ConditionFalse), fakeCondition(v1alpha1.RootSyncStalled, metav1.ConditionTrue))),
			fakeConditionMessage,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := StalledMessage(tc.rs)
			if got != tc.want {
				t.Errorf("got StalledMessage() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestClearCondition(t *testing.T) {
	now = func() metav1.Time {
		return testNow
	}
	testCases := []struct {
		name    string
		rs      *v1alpha1.RootSync
		toClear v1alpha1.RootSyncConditionType
		want    []v1alpha1.RootSyncCondition
	}{
		{
			"Clear existing true condition",
			fake.RootSyncObject(withConditions(fakeCondition(v1alpha1.RootSyncReconciling, metav1.ConditionTrue), fakeCondition(v1alpha1.RootSyncStalled, metav1.ConditionTrue))),
			v1alpha1.RootSyncStalled,
			[]v1alpha1.RootSyncCondition{
				fakeCondition(v1alpha1.RootSyncReconciling, metav1.ConditionTrue),
				fakeCondition(v1alpha1.RootSyncStalled, metav1.ConditionFalse, "", ""),
			},
		},
		{
			"Ignore existing false condition",
			fake.RootSyncObject(withConditions(fakeCondition(v1alpha1.RootSyncReconciling, metav1.ConditionTrue), fakeCondition(v1alpha1.RootSyncStalled, metav1.ConditionFalse))),
			v1alpha1.RootSyncStalled,
			[]v1alpha1.RootSyncCondition{
				fakeCondition(v1alpha1.RootSyncReconciling, metav1.ConditionTrue),
				fakeCondition(v1alpha1.RootSyncStalled, metav1.ConditionFalse),
			},
		},
		{
			"Handle empty conditions",
			fake.RootSyncObject(),
			v1alpha1.RootSyncStalled,
			nil,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ClearCondition(tc.rs, tc.toClear)
			if diff := cmp.Diff(tc.want, tc.rs.Status.Conditions); diff != "" {
				t.Error(diff)
			}
		})
	}
}

func TestSetReconciling(t *testing.T) {
	now = func() metav1.Time {
		return testNow
	}
	testCases := []struct {
		name    string
		rs      *v1alpha1.RootSync
		reason  string
		message string
		want    []v1alpha1.RootSyncCondition
	}{
		{
			"Set new reconciling condition",
			fake.RootSyncObject(),
			"Test1",
			"This is test 1",
			[]v1alpha1.RootSyncCondition{
				fakeCondition(v1alpha1.RootSyncReconciling, metav1.ConditionTrue, "Test1", "This is test 1"),
			},
		},
		{
			"Update existing reconciling condition",
			fake.RootSyncObject(withConditions(fakeCondition(v1alpha1.RootSyncReconciling, metav1.ConditionFalse), fakeCondition(v1alpha1.RootSyncStalled, metav1.ConditionFalse))),
			"Test2",
			"This is test 2",
			[]v1alpha1.RootSyncCondition{
				fakeCondition(v1alpha1.RootSyncReconciling, metav1.ConditionTrue, "Test2", "This is test 2"),
				fakeCondition(v1alpha1.RootSyncStalled, metav1.ConditionFalse),
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			SetReconciling(tc.rs, tc.reason, tc.message)
			if diff := cmp.Diff(tc.want, tc.rs.Status.Conditions); diff != "" {
				t.Error(diff)
			}
		})
	}
}

func TestSetStalled(t *testing.T) {
	testCases := []struct {
		name   string
		rs     *v1alpha1.RootSync
		reason string
		err    error
		want   []v1alpha1.RootSyncCondition
	}{
		{
			"Set new stalled condition",
			fake.RootSyncObject(),
			"Error1",
			errors.New("this is error 1"),
			[]v1alpha1.RootSyncCondition{
				fakeCondition(v1alpha1.RootSyncStalled, metav1.ConditionTrue, "Error1", "this is error 1"),
			},
		},
		{
			"Update existing stalled condition",
			fake.RootSyncObject(withConditions(fakeCondition(v1alpha1.RootSyncReconciling, metav1.ConditionTrue), fakeCondition(v1alpha1.RootSyncStalled, metav1.ConditionFalse))),
			"Error2",
			errors.New("this is error 2"),
			[]v1alpha1.RootSyncCondition{
				fakeCondition(v1alpha1.RootSyncReconciling, metav1.ConditionTrue),
				fakeCondition(v1alpha1.RootSyncStalled, metav1.ConditionTrue, "Error2", "this is error 2"),
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			SetStalled(tc.rs, tc.reason, tc.err)
			if diff := cmp.Diff(tc.want, tc.rs.Status.Conditions); diff != "" {
				t.Error(diff)
			}
		})
	}
}
