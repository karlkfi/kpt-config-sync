package watch

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/testing/fake"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/rest"
)

func fakeRunnable(opts watcherOptions) (Runnable, error) {
	return &filteredWatcher{
		base:      watch.NewFake(),
		resources: nil,
		queue:     nil,
	}, nil
}

func TestManager(t *testing.T) {
	var config *rest.Config
	options := &Options{
		watcherFunc: fakeRunnable,
	}
	m, err := NewManager(config, nil, options)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}

	obj1 := fake.DeploymentObject()
	obj2 := fake.ConfigMapObject()
	objects := []core.Object{obj1, obj2}

	err = m.Update(objects)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}

	want := map[schema.GroupVersionKind]Runnable{
		obj1.GroupVersionKind(): nil,
		obj2.GroupVersionKind(): nil,
	}

	compare(t, want, m.watcherMap)

	deploymentOtherVersion := obj1.DeepCopy()
	deploymentOtherVersion.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "apps",
		Version: "v1beta1",
		Kind:    "Deployment",
	})
	objects = []core.Object{obj1, obj2, deploymentOtherVersion}
	err = m.Update(objects)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}

	compare(t, want, m.watcherMap)

	objects = []core.Object{}
	err = m.Update(objects)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}

	var emptyWatchers = map[schema.GroupVersionKind]Runnable{}
	if diff := cmp.Diff(emptyWatchers, m.watcherMap, cmpopts.EquateEmpty()); diff != "" {
		t.Errorf("not all watchers were stopped:\n%s", diff)
	}
}

func TestManagerDiffererntVersions(t *testing.T) {
	var config *rest.Config
	options := &Options{
		watcherFunc: fakeRunnable,
	}
	m, err := NewManager(config, nil, options)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}

	deploymentv1 := fake.DeploymentObject()
	deploymentv1beta1 := deploymentv1.DeepCopy()
	deploymentv1beta1.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "apps",
		Version: "v1beta1",
		Kind:    "Deployment",
	})

	objects := []core.Object{deploymentv1}
	err = m.Update(objects)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}

	objects = []core.Object{deploymentv1beta1}
	err = m.Update(objects)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}

	want := map[schema.GroupVersionKind]Runnable{
		deploymentv1.GroupVersionKind(): nil,
	}
	compare(t, want, m.watcherMap)

	// update the manager to stop all watchers
	objects = []core.Object{}
	err = m.Update(objects)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}

	objects = []core.Object{deploymentv1beta1}
	err = m.Update(objects)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}

	objects = []core.Object{deploymentv1}
	err = m.Update(objects)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}

	want = map[schema.GroupVersionKind]Runnable{
		deploymentv1beta1.GroupVersionKind(): nil,
	}

	compare(t, want, m.watcherMap)
}

func compare(t *testing.T, a, b map[schema.GroupVersionKind]Runnable) {
	if len(a) != len(b) {
		t.Errorf("%v and %v don't have the same size", a, b)
	}
	for key := range a {
		if _, found := b[key]; !found {
			t.Errorf("%v not found in %v", key, b)
		}
	}
}
