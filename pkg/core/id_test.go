package core_test

import (
	"testing"

	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/testing/fake"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestGKNN(t *testing.T) {
	testcases := []struct {
		name string
		obj  client.Object
		want string
	}{
		{
			name: "a namespaced object",
			obj:  fake.RoleObject(core.Namespace("test")),
			want: "rbac.authorization.k8s.io_role_test_default-name",
		},
		{
			name: "a cluster-scoped object",
			obj:  fake.ClusterRoleObject(),
			want: "rbac.authorization.k8s.io_clusterrole_default-name",
		},
		{
			name: "a nil object",
			obj:  nil,
			want: "",
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			got := core.GKNN(tc.obj)
			if tc.want != got {
				t.Errorf("GKNN() = %q, got %q", got, tc.want)
			}
		})
	}
}
