package parse

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/testing/fake"
)

func TestSortByScope(t *testing.T) {
	testCases := []struct {
		name string
		objs []core.Object
		want []core.Object
	}{
		{
			name: "Empty list",
			objs: []core.Object{},
			want: []core.Object{},
		},
		{
			name: "Needs sorting",
			objs: []core.Object{
				fake.RoleObject(core.Namespace("foo")),
				fake.ResourceQuotaObject(core.Namespace("foo")),
				fake.ClusterRoleObject(),
				fake.NamespaceObject("foo"),
			},
			want: []core.Object{
				fake.ClusterRoleObject(),
				fake.NamespaceObject("foo"),
				fake.RoleObject(core.Namespace("foo")),
				fake.ResourceQuotaObject(core.Namespace("foo")),
			},
		},
		{
			name: "Already sorted",
			objs: []core.Object{
				fake.NamespaceObject("foo"),
				fake.RoleObject(core.Namespace("foo")),
				fake.ResourceQuotaObject(core.Namespace("foo")),
			},
			want: []core.Object{
				fake.NamespaceObject("foo"),
				fake.RoleObject(core.Namespace("foo")),
				fake.ResourceQuotaObject(core.Namespace("foo")),
			},
		},
		{
			name: "Only namespace scoped",
			objs: []core.Object{
				fake.RoleObject(core.Namespace("foo")),
				fake.ResourceQuotaObject(core.Namespace("foo")),
			},
			want: []core.Object{
				fake.RoleObject(core.Namespace("foo")),
				fake.ResourceQuotaObject(core.Namespace("foo")),
			},
		},
		{
			name: "Only cluster scoped",
			objs: []core.Object{
				fake.ClusterRoleObject(),
				fake.NamespaceObject("foo"),
			},
			want: []core.Object{
				fake.ClusterRoleObject(),
				fake.NamespaceObject("foo"),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			objs := make([]core.Object, len(tc.objs))
			copy(objs, tc.objs)
			sortByScope(objs)
			if diff := cmp.Diff(objs, tc.want); diff != "" {
				t.Error(diff)
			}
		})
	}
}
