package reconcile

import (
	"context"
	"testing"

	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/remediator/queue"
	syncertesting "github.com/google/nomos/pkg/syncer/testing"
	"github.com/google/nomos/pkg/testing/fake"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestWorker_ProcessNextObject(t *testing.T) {
	testCases := []struct {
		name      string
		declared  []core.Object
		toProcess []core.Object
		want      []runtime.Object
	}{
		{
			name: "update actual objects",
			declared: []core.Object{
				fake.ClusterRoleBindingObject(core.Label("first", "one")),
				fake.ClusterRoleObject(core.Label("second", "two")),
			},
			toProcess: []core.Object{
				fake.ClusterRoleBindingObject(syncertesting.ManagementEnabled),
				fake.ClusterRoleObject(syncertesting.ManagementEnabled),
			},
			want: []runtime.Object{
				// TODO(b/162547054): Figure out why the reconciler is stripping away labels and annotations.
				fake.ClusterRoleBindingObject(core.Label("first", "one")),
				fake.ClusterRoleObject(core.Label("second", "two")),
			},
		},
		{
			name:     "delete undeclared objects",
			declared: []core.Object{},
			toProcess: []core.Object{
				fake.ClusterRoleBindingObject(syncertesting.ManagementEnabled),
				fake.ClusterRoleObject(syncertesting.ManagementEnabled),
			},
			want: []runtime.Object{},
		},
		{
			name: "create missing objects",
			declared: []core.Object{
				fake.ClusterRoleBindingObject(),
				fake.ClusterRoleObject(),
			},
			toProcess: []core.Object{},
			want:      []runtime.Object{
				// TODO(b/159821780): Update the workqueue to properly funnel delete events to the reconciler.
				//fake.ClusterRoleBindingObject(),
				//fake.ClusterRoleObject(),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			q := queue.NewNamed("test")
			for _, obj := range tc.toProcess {
				q.Add(obj)
			}

			c := fakeClient(t, tc.toProcess...)
			d := declared(t, tc.declared...)
			w := NewWorker(c.Applier(), q, d)

			for _, obj := range tc.toProcess {
				if ok := w.processNextObject(context.Background()); !ok {
					t.Errorf("unexpected false result from processNextObject() for object: %v", obj)
				}
			}

			c.Check(t, tc.want...)
		})
	}
}
