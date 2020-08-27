package diff

import (
	"testing"

	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/declared"
	"github.com/google/nomos/pkg/diff/difftest"
	"github.com/google/nomos/pkg/syncer/syncertest"
	"github.com/google/nomos/pkg/testing/fake"
)

func TestCanManage(t *testing.T) {
	testCases := []struct {
		name       string
		reconciler declared.Scope
		object     core.Object
		want       bool
	}{
		{
			"Root can manage unmanaged object",
			declared.RootReconciler,
			fake.DeploymentObject(),
			true,
		},
		{
			"Root can manage other-managed object",
			declared.RootReconciler,
			fake.DeploymentObject(syncertest.ManagementEnabled, difftest.ManagedBy("foo")),
			true,
		},
		{
			"Root can manage self-managed object",
			declared.RootReconciler,
			fake.DeploymentObject(syncertest.ManagementEnabled, difftest.ManagedByRoot),
			true,
		},
		{
			"Non-root can manage unmanaged object",
			"foo",
			fake.DeploymentObject(),
			true,
		},
		{
			"Non-root can manage self-managed object",
			"foo",
			fake.DeploymentObject(syncertest.ManagementEnabled, difftest.ManagedBy("foo")),
			true,
		},
		{
			"Non-root can manage other-managed object",
			"foo",
			fake.DeploymentObject(syncertest.ManagementEnabled, difftest.ManagedBy("foo")),
			true,
		},
		{
			"Non-root can NOT manage root-managed object",
			"foo",
			fake.DeploymentObject(syncertest.ManagementEnabled, difftest.ManagedByRoot),
			false,
		},
		{
			"Non-root can manage seemingly root-managed object",
			"foo",
			fake.DeploymentObject(difftest.ManagedByRoot),
			true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := CanManage(tc.reconciler, tc.object)
			if got != tc.want {
				t.Errorf("CanManage() = %v; want %v", got, tc.want)
			}
		})
	}
}
