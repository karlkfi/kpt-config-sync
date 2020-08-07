package declared

import (
	"testing"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/testing/fake"
)

func TestCanManage(t *testing.T) {
	testCases := []struct {
		name       string
		reconciler string
		object     core.Object
		want       bool
	}{
		{
			"Root can manage unmanaged object",
			RootReconciler,
			fake.DeploymentObject(),
			true,
		},
		{
			"Root can manage other-managed object",
			RootReconciler,
			fake.DeploymentObject(core.Annotation(v1.ResourceManagementKey, v1.ResourceManagementEnabled), core.Annotation(v1.ResourceManagerKey, "foo")),
			true,
		},
		{
			"Root can manage self-managed object",
			RootReconciler,
			fake.DeploymentObject(core.Annotation(v1.ResourceManagementKey, v1.ResourceManagementEnabled), core.Annotation(v1.ResourceManagerKey, RootReconciler)),
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
			fake.DeploymentObject(core.Annotation(v1.ResourceManagementKey, v1.ResourceManagementEnabled), core.Annotation(v1.ResourceManagerKey, "foo")),
			true,
		},
		{
			"Non-root can manage other-managed object",
			"foo",
			fake.DeploymentObject(core.Annotation(v1.ResourceManagementKey, v1.ResourceManagementEnabled), core.Annotation(v1.ResourceManagerKey, "bar")),
			true,
		},
		{
			"Non-root can NOT manage root-managed object",
			"foo",
			fake.DeploymentObject(core.Annotation(v1.ResourceManagementKey, v1.ResourceManagementEnabled), core.Annotation(v1.ResourceManagerKey, RootReconciler)),
			false,
		},
		{
			"Non-root can manage seemingly root-managed object",
			"foo",
			fake.DeploymentObject(core.Annotation(v1.ResourceManagerKey, RootReconciler)),
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
