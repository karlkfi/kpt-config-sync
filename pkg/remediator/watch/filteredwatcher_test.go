package watch

import (
	"testing"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/parse/declaredresources"
	"github.com/google/nomos/pkg/testing/fake"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/watch"
)

type fakeQueue struct {
	m map[core.ID]core.Object
}

func (q *fakeQueue) Add(obj core.Object) {
	q.m[core.IDOf(obj)] = obj
}

func prepareObjts() (u1, u2, u3 *unstructured.Unstructured) {
	// an object that can be found in the declared resources
	u1 = fake.UnstructuredObject(kinds.Deployment(),
		core.Name("default-name"),
		core.Namespace("default"))

	// an object that can't be found in declared resources
	// and isn't managed by config sync
	u2 = fake.UnstructuredObject(kinds.Deployment(),
		core.Name("unwatched"),
		core.Namespace("default"))

	// an object that is managed by config sync
	u3 = fake.UnstructuredObject(kinds.Deployment(),
		core.Name("managed-resource"),
		core.Namespace("default"))
	u3.SetAnnotations(
		map[string]string{
			v1.ResourceManagementKey: v1.ResourceManagementEnabled,
		},
	)
	return
}

func TestWrappedWatcher(t *testing.T) {
	u1, u2, u3 := prepareObjts()
	obj, err := core.ObjectOf(u1)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}

	resources := declaredresources.NewDeclaredResources()
	err = resources.UpdateDecls([]core.Object{obj})
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}

	base := watch.NewFakeWithChanSize(3, false)
	base.Add(u1)
	base.Add(u2)
	base.Add(u3)

	q := &fakeQueue{m: make(map[core.ID]core.Object)}
	w := filteredWatcher{
		resources: resources,
		base:      base,
		queue:     q,
	}

	w.Stop()
	w.Run()

	if len(q.m) != 2 {
		t.Fatalf("fake queue should contain two objects, but got %d", len(q.m))
	}

	for _, u := range []*unstructured.Unstructured{u1, u3} {
		id := core.IDOfUnstructured(*u)
		if _, found := q.m[id]; !found {
			t.Errorf("%v should be in the queue", id)
		}
	}
}
