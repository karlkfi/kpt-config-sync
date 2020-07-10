package watch

import (
	"sync"
	"testing"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/parse/declaredresources"
	"github.com/google/nomos/pkg/testing/fake"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type fakeQueue struct {
	l sync.RWMutex
	m map[GVKNN]struct{}
}

func (q *fakeQueue) Add(gvknn GVKNN) {
	q.l.Lock()
	q.m[gvknn] = struct{}{}
	q.l.Unlock()
}

func setup(t *testing.T, objs ...core.Object) (*declaredresources.DeclaredResources, *fakeQueue) {
	resources := declaredresources.NewDeclaredResources()
	err := resources.UpdateDecls(objs)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}

	queue := &fakeQueue{
		m: make(map[GVKNN]struct{}),
	}
	return resources, queue
}

func TestIgnoreUnwatchedObject(t *testing.T) {
	queue := &fakeQueue{
		m: make(map[GVKNN]struct{}),
	}
	h := handler{
		resources: declaredresources.NewDeclaredResources(),
		queue:     queue,
	}

	obj := fake.DeploymentObject()
	h.OnAdd(obj)
	h.OnDelete(obj)
	h.OnUpdate(obj, obj)
	if len(queue.m) != 0 {
		t.Fatal("the event shouldn't be added to the queue")
	}
}

func TestOneHandler(t *testing.T) {
	obj1 := fake.DeploymentObject()
	obj2 := fake.ConfigMapObject()
	resources, queue := setup(t, obj1, obj2)
	h := handler{
		resources: resources,
		queue:     queue,
	}
	obj := fake.DeploymentObject()
	h.OnAdd(obj)
	if len(queue.m) != 1 {
		t.Fatalf("failed to add event to the queue")
	}
	id, err := core.IDOfRuntime(obj)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	if _, found := queue.m[GVKNN{ID: id, Version: obj.GroupVersionKind().Version}]; !found {
		t.Fatalf("unable to find the expected event in the queue")
	}
}

func TestMultipleHandlers(t *testing.T) {
	obj1 := fake.DeploymentObject()
	obj2 := fake.ConfigMapObject()
	resources, queue := setup(t, obj1, obj2)
	h1 := handler{
		resources: resources,
		queue:     queue,
	}
	h2 := handler{
		resources: resources,
		queue:     queue,
	}

	worker := func(id int, wg *sync.WaitGroup) {
		defer wg.Done()
		if id == 1 {
			h1.OnAdd(obj1)
		}
		if id == 2 {
			h2.OnDelete(obj2)
		}
	}

	var wg sync.WaitGroup
	for i := 1; i <= 2; i++ {
		wg.Add(1)
		go worker(i, &wg)
	}
	wg.Wait()

	if len(queue.m) != 2 {
		t.Fatalf("failed to add event to the queue")
	}

	// Verify the events in the queue are as expected
	for _, obj := range []core.Object{obj1, obj2} {
		id, err := core.IDOfRuntime(obj)
		if err != nil {
			t.Fatalf("unexpected error %v", err)
		}
		if _, found := queue.m[GVKNN{ID: id, Version: obj.GroupVersionKind().Version}]; !found {
			t.Fatalf("unable to find the expected event in the queue")
		}
	}
}

func TestManagedResource(t *testing.T) {
	obj := fake.DeploymentObject()
	obj.SetAnnotations(
		map[string]string{
			v1.ResourceManagementKey: v1.ResourceManagementEnabled,
		})
	// the declared resources don't contain any resources
	resources, queue := setup(t)
	h := handler{
		resources: resources,
		queue:     queue,
	}

	h.OnAdd(obj)
	if len(queue.m) != 1 {
		t.Fatalf("failed to add event to the queue")
	}

	// Verify the events in the queue are as expected
	id, err := core.IDOfRuntime(obj)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	if _, found := queue.m[GVKNN{ID: id, Version: obj.GroupVersionKind().Version}]; !found {
		t.Fatalf("unable to find the expected event in the queue")
	}
}

func TestNoObjectMeta(t *testing.T) {
	obj := &corev1.ConfigMapList{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMapList",
		},
	}

	// the declared resources don't contain any resources
	resources, queue := setup(t)
	h := handler{
		resources: resources,
		queue:     queue,
	}

	h.OnAdd(obj)
	// Verify no event is in the queue
	if len(queue.m) != 0 {
		t.Fatalf("there shouldn't be any event in the queue")
	}
}
