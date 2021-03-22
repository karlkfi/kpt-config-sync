package reposync

import (
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/testing/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const fakeConditionMessage = "Testing"

var testNow = metav1.Date(1, time.February, 3, 4, 5, 6, 7, time.Local)

func withConditions(conds ...v1alpha1.RepoSyncCondition) core.MetaMutator {
	return func(o client.Object) {
		rs := o.(*v1alpha1.RepoSync)
		rs.Status.Conditions = append(rs.Status.Conditions, conds...)
	}
}

func fakeCondition(condType v1alpha1.RepoSyncConditionType, status metav1.ConditionStatus, strs ...string) v1alpha1.RepoSyncCondition {
	rsc := v1alpha1.RepoSyncCondition{
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
		rs   *v1alpha1.RepoSync
		want bool
	}{
		{
			"Missing condition is false",
			fake.RepoSyncObject(),
			false,
		},
		{
			"False condition is false",
			fake.RepoSyncObject(withConditions(fakeCondition(v1alpha1.RepoSyncReconciling, metav1.ConditionFalse))),
			false,
		},
		{
			"True condition is true",
			fake.RepoSyncObject(withConditions(fakeCondition(v1alpha1.RepoSyncReconciling, metav1.ConditionTrue), fakeCondition(v1alpha1.RepoSyncStalled, metav1.ConditionFalse))),
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
		rs   *v1alpha1.RepoSync
		want bool
	}{
		{
			"Missing condition is false",
			fake.RepoSyncObject(),
			false,
		},
		{
			"False condition is false",
			fake.RepoSyncObject(withConditions(fakeCondition(v1alpha1.RepoSyncStalled, metav1.ConditionFalse))),
			false,
		},
		{
			"True condition is true",
			fake.RepoSyncObject(withConditions(fakeCondition(v1alpha1.RepoSyncReconciling, metav1.ConditionFalse), fakeCondition(v1alpha1.RepoSyncStalled, metav1.ConditionTrue))),
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
		rs   *v1alpha1.RepoSync
		want string
	}{
		{
			"Missing condition is empty",
			fake.RepoSyncObject(),
			"",
		},
		{
			"False condition is empty",
			fake.RepoSyncObject(withConditions(fakeCondition(v1alpha1.RepoSyncReconciling, metav1.ConditionFalse))),
			"",
		},
		{
			"True condition is its message",
			fake.RepoSyncObject(withConditions(fakeCondition(v1alpha1.RepoSyncReconciling, metav1.ConditionTrue), fakeCondition(v1alpha1.RepoSyncStalled, metav1.ConditionFalse))),
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
		rs   *v1alpha1.RepoSync
		want string
	}{
		{
			"Missing condition is empty",
			fake.RepoSyncObject(),
			"",
		},
		{
			"False condition is empty",
			fake.RepoSyncObject(withConditions(fakeCondition(v1alpha1.RepoSyncStalled, metav1.ConditionFalse))),
			"",
		},
		{
			"True condition is its message",
			fake.RepoSyncObject(withConditions(fakeCondition(v1alpha1.RepoSyncReconciling, metav1.ConditionFalse), fakeCondition(v1alpha1.RepoSyncStalled, metav1.ConditionTrue))),
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
		rs      *v1alpha1.RepoSync
		toClear v1alpha1.RepoSyncConditionType
		want    []v1alpha1.RepoSyncCondition
	}{
		{
			"Clear existing true condition",
			fake.RepoSyncObject(withConditions(fakeCondition(v1alpha1.RepoSyncReconciling, metav1.ConditionTrue), fakeCondition(v1alpha1.RepoSyncStalled, metav1.ConditionTrue))),
			v1alpha1.RepoSyncStalled,
			[]v1alpha1.RepoSyncCondition{
				fakeCondition(v1alpha1.RepoSyncReconciling, metav1.ConditionTrue),
				fakeCondition(v1alpha1.RepoSyncStalled, metav1.ConditionFalse, "", ""),
			},
		},
		{
			"Ignore existing false condition",
			fake.RepoSyncObject(withConditions(fakeCondition(v1alpha1.RepoSyncReconciling, metav1.ConditionTrue), fakeCondition(v1alpha1.RepoSyncStalled, metav1.ConditionFalse))),
			v1alpha1.RepoSyncStalled,
			[]v1alpha1.RepoSyncCondition{
				fakeCondition(v1alpha1.RepoSyncReconciling, metav1.ConditionTrue),
				fakeCondition(v1alpha1.RepoSyncStalled, metav1.ConditionFalse),
			},
		},
		{
			"Handle empty conditions",
			fake.RepoSyncObject(),
			v1alpha1.RepoSyncStalled,
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
		rs      *v1alpha1.RepoSync
		reason  string
		message string
		want    []v1alpha1.RepoSyncCondition
	}{
		{
			"Set new reconciling condition",
			fake.RepoSyncObject(),
			"Test1",
			"This is test 1",
			[]v1alpha1.RepoSyncCondition{
				fakeCondition(v1alpha1.RepoSyncReconciling, metav1.ConditionTrue, "Test1", "This is test 1"),
			},
		},
		{
			"Update existing reconciling condition",
			fake.RepoSyncObject(withConditions(fakeCondition(v1alpha1.RepoSyncReconciling, metav1.ConditionFalse), fakeCondition(v1alpha1.RepoSyncStalled, metav1.ConditionFalse))),
			"Test2",
			"This is test 2",
			[]v1alpha1.RepoSyncCondition{
				fakeCondition(v1alpha1.RepoSyncReconciling, metav1.ConditionTrue, "Test2", "This is test 2"),
				fakeCondition(v1alpha1.RepoSyncStalled, metav1.ConditionFalse),
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
		rs     *v1alpha1.RepoSync
		reason string
		err    error
		want   []v1alpha1.RepoSyncCondition
	}{
		{
			"Set new stalled condition",
			fake.RepoSyncObject(),
			"Error1",
			errors.New("this is error 1"),
			[]v1alpha1.RepoSyncCondition{
				fakeCondition(v1alpha1.RepoSyncStalled, metav1.ConditionTrue, "Error1", "this is error 1"),
			},
		},
		{
			"Update existing stalled condition",
			fake.RepoSyncObject(withConditions(fakeCondition(v1alpha1.RepoSyncReconciling, metav1.ConditionTrue), fakeCondition(v1alpha1.RepoSyncStalled, metav1.ConditionFalse))),
			"Error2",
			errors.New("this is error 2"),
			[]v1alpha1.RepoSyncCondition{
				fakeCondition(v1alpha1.RepoSyncReconciling, metav1.ConditionTrue),
				fakeCondition(v1alpha1.RepoSyncStalled, metav1.ConditionTrue, "Error2", "this is error 2"),
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
