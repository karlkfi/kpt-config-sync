package watch

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/parse/declaredresources"
	"github.com/google/nomos/pkg/remediator/queue"
	"github.com/google/nomos/pkg/testing/fake"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
)

type action struct {
	event watch.EventType
	obj   runtime.Object
}

func TestWrappedWatcher(t *testing.T) {
	deployment1 := fake.DeploymentObject(core.Name("hello"))
	deployment1Beta := fake.DeploymentObject(core.Name("hello"))
	deployment1Beta.GetObjectKind().SetGroupVersionKind(deployment1Beta.GroupVersionKind().GroupKind().WithVersion("beta1"))

	deployment2 := fake.DeploymentObject(core.Name("world"))
	deployment3 := fake.DeploymentObject(core.Name("nomes"))
	managedDeployment := fake.DeploymentObject(core.Name("not-declared"), core.Annotation(v1.ResourceManagementKey, v1.ResourceManagementEnabled))

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
			"Enqueue events for undeclared-but-managed resource",
			[]core.Object{
				deployment1,
			},
			[]action{
				{
					watch.Modified,
					managedDeployment,
				},
			},
			[]core.ID{
				core.IDOf(managedDeployment),
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
			"Ignore bookmark events",
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
					nil,
				},
			},
			[]core.ID{
				core.IDOf(deployment1),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dr := declaredresources.NewDeclaredResources()
			if err := dr.UpdateDecls(tc.declared); err != nil {
				t.Fatalf("unexpected error %v", err)
			}

			base := watch.NewFakeWithChanSize(len(tc.actions), false)
			for _, a := range tc.actions {
				base.Action(a.event, a.obj)
			}

			q := queue.NewNamed("test")
			w := filteredWatcher{
				resources: dr,
				base:      base,
				queue:     q,
			}

			w.Stop()
			w.Run()

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
