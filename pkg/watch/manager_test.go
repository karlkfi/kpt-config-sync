package watch

import (
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"

	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/testing/fake"
)

func fakeInformer(schema.GroupVersionKind, meta.RESTMapper, *rest.Config,
	time.Duration) (mapEntry, error) {
	return mapEntry{stopCh: make(chan struct{})}, nil
}

func TestManager(t *testing.T) {
	var config *rest.Config
	options := &Options{
		Resync:       time.Hour,
		InformerFunc: fakeInformer,
	}
	m, err := NewManager(config, options)
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

	if _, ok := m.informers[obj1.GroupVersionKind()]; !ok {
		t.Errorf("informer not started for %v", obj1.GroupVersionKind())
	}
	if _, ok := m.informers[obj2.GroupVersionKind()]; !ok {
		t.Errorf("informer not started for %v", obj2.GroupVersionKind())
	}

	obj3 := obj1.DeepCopy()
	obj3.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "apps",
		Version: "v1beta1",
		Kind:    "Deployment",
	})
	objects = []core.Object{obj1, obj2, obj3}
	err = m.Update(objects)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	if _, ok := m.informers[obj1.GroupVersionKind()]; !ok {
		t.Errorf("informer not started for %v", obj1.GroupVersionKind())
	}
	if _, ok := m.informers[obj2.GroupVersionKind()]; !ok {
		t.Errorf("informer not started for %v", obj2.GroupVersionKind())
	}
	if _, ok := m.informers[obj3.GroupVersionKind()]; ok {
		t.Errorf("there shouldn't be an informer for another version of Deployment")
	}

	objects = []core.Object{}
	err = m.Update(objects)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}

	var emptyInformers = map[schema.GroupVersionKind]mapEntry{}
	if diff := cmp.Diff(emptyInformers, m.informers, cmpopts.EquateEmpty()); diff != "" {
		t.Errorf("not all informers were stopped:\n%s", diff)
	}
}
