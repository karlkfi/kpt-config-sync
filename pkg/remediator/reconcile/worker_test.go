package reconcile

import (
	"context"
	"testing"

	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/declared"
	"github.com/google/nomos/pkg/remediator/queue"
	"github.com/google/nomos/pkg/syncer/syncertest"
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
				fake.ClusterRoleBindingObject(syncertest.ManagementEnabled,
					core.Label("first", "one")),
				fake.ClusterRoleObject(syncertest.ManagementEnabled,
					core.Label("second", "two")),
			},
			toProcess: []core.Object{
				fake.ClusterRoleBindingObject(syncertest.ManagementEnabled),
				fake.ClusterRoleObject(syncertest.ManagementEnabled),
			},
			want: []runtime.Object{
				// TODO(b/162547054): Figure out why the reconciler is stripping away labels and annotations.
				fake.ClusterRoleBindingObject(syncertest.ManagementEnabled,
					core.Label("first", "one")),
				fake.ClusterRoleObject(syncertest.ManagementEnabled,
					core.Label("second", "two")),
			},
		},
		{
			name:     "delete undeclared objects",
			declared: []core.Object{},
			toProcess: []core.Object{
				fake.ClusterRoleBindingObject(syncertest.ManagementEnabled),
				fake.ClusterRoleObject(syncertest.ManagementEnabled),
			},
			want: []runtime.Object{},
		},
		{
			name: "create missing objects",
			declared: []core.Object{
				fake.ClusterRoleBindingObject(syncertest.ManagementEnabled),
				fake.ClusterRoleObject(syncertest.ManagementEnabled),
			},
			toProcess: []core.Object{
				queue.MarkDeleted(fake.ClusterRoleBindingObject()),
				queue.MarkDeleted(fake.ClusterRoleObject()),
			},
			want: []runtime.Object{
				fake.ClusterRoleBindingObject(syncertest.ManagementEnabled),
				fake.ClusterRoleObject(syncertest.ManagementEnabled),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			q := queue.New("test")
			for _, obj := range tc.toProcess {
				q.Add(obj)
			}

			c := fakeClient(t)
			for _, obj := range tc.toProcess {
				if !queue.WasDeleted(obj) {
					if err := c.Create(context.Background(), obj); err != nil {
						t.Fatalf("Failed to create object in fake client: %v", err)
					}
				}
			}

			d := makeDeclared(t, tc.declared...)
			w := NewWorker(declared.RootReconciler, c.Applier(), q, d)

			for _, obj := range tc.toProcess {
				if ok := w.processNextObject(context.Background()); !ok {
					t.Errorf("unexpected false result from processNextObject() for object: %v", obj)
				}
			}

			c.Check(t, tc.want...)
		})
	}
}
