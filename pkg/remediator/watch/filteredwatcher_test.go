package watch

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/declared"
	"github.com/google/nomos/pkg/diff/difftest"
	"github.com/google/nomos/pkg/remediator/queue"
	"github.com/google/nomos/pkg/syncer/syncertest"
	"github.com/google/nomos/pkg/testing/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
)

type action struct {
	event watch.EventType
	obj   runtime.Object
}

func TestFilteredWatcher(t *testing.T) {
	reconciler := declared.Scope("test")

	deployment1 := fake.DeploymentObject(core.Name("hello"))
	deployment1Beta := fake.DeploymentObject(core.Name("hello"))
	deployment1Beta.GetObjectKind().SetGroupVersionKind(deployment1Beta.GroupVersionKind().GroupKind().WithVersion("beta1"))

	deployment2 := fake.DeploymentObject(core.Name("world"))
	deployment3 := fake.DeploymentObject(core.Name("nomes"))

	managedBySelfDeployment := fake.DeploymentObject(core.Name("not-declared"),
		syncertest.ManagementEnabled, difftest.ManagedBy(reconciler))
	managedByOtherDeployment := fake.DeploymentObject(core.Name("not-declared"),
		syncertest.ManagementEnabled, difftest.ManagedBy("other"))
	deploymentForRoot := fake.DeploymentObject(core.Name("managed-by-root"), difftest.ManagedByRoot)

	testCases := []struct {
		name     string
		declared []core.Object
		actions  []action
		want     []core.ID
	}{
		{
			"Enqueue events for declared resources",
			[]core.Object{
				deployment1,
				deployment2,
				deployment3,
			},
			[]action{
				{
					watch.Added,
					deployment1,
				},
				{
					watch.Modified,
					deployment2,
				},
				{
					watch.Deleted,
					deployment3,
				},
			},
			[]core.ID{
				core.IDOf(deployment1),
				core.IDOf(deployment2),
				core.IDOf(deployment3),
			},
		},
		{
			"Enqueue events for undeclared-but-managed-by-other-reconciler resource",
			[]core.Object{
				deployment1,
			},
			[]action{
				{
					watch.Modified,
					managedByOtherDeployment,
				},
			},
			nil,
		},
		{
			"Enqueue events for undeclared-but-managed-by-this-reconciler resource",
			[]core.Object{
				deployment1,
			},
			[]action{
				{
					watch.Modified,
					managedBySelfDeployment,
				},
			},
			[]core.ID{
				core.IDOf(managedBySelfDeployment),
			},
		},
		{
			"Filter events for undeclared-and-unmanaged resources",
			[]core.Object{
				deployment1,
			},
			[]action{
				{
					watch.Added,
					deployment2,
				},
				{
					watch.Added,
					deployment3,
				},
			},
			nil,
		},
		{
			"Filter events for declared resource with different manager",
			[]core.Object{
				deployment1,
			},
			[]action{
				{
					watch.Modified,
					deploymentForRoot,
				},
			},
			nil,
		},
		{
			"Filter events for declared resource with different GVK",
			[]core.Object{
				deployment1,
			},
			[]action{
				{
					watch.Modified,
					deployment1Beta,
				},
			},
			nil,
		},
		{
			"Handle bookmark events",
			[]core.Object{
				deployment1,
			},
			[]action{
				{
					watch.Modified,
					deployment1,
				},
				{
					watch.Bookmark,
					deployment1,
				},
			},
			[]core.ID{
				core.IDOf(deployment1),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dr := &declared.Resources{}
			if err := dr.Update(tc.declared); err != nil {
				t.Fatalf("unexpected error %v", err)
			}

			base := watch.NewFake()
			q := queue.New("test")
			opts := watcherOptions{
				reconciler: reconciler,
				resources:  dr,
				queue:      q,
				startWatch: func(options metav1.ListOptions) (watch.Interface, error) {
					return base, nil
				},
			}
			w := NewFiltered(opts)

			go func() {
				for _, a := range tc.actions {
					// Each base.Action() blocks until the code within w.Run() reads its
					// event from the queue.
					base.Action(a.event, a.obj)
				}
				// This is not reached until after w.Run() reads the last event from the
				// queue.
				w.Stop()
			}()
			// w.Run() blocks until w.Stop() is called.
			if err := w.Run(); err != nil {
				t.Fatalf("got Run() = %v, want Run() = <nil>", err)
			}

			var got []core.ID
			for q.Len() > 0 {
				obj, shutdown := q.Get()
				if shutdown {
					t.Fatal("Object queue was shut down unexpectedly.")
				}
				got = append(got, core.IDOf(obj))
			}

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("did not get desired object IDs: %v", diff)
			}
		})
	}
}
