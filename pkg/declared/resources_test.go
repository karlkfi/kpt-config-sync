package declared

import (
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/syncer/reconcile"
	"github.com/google/nomos/pkg/testing/fake"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	obj1 = fake.CustomResourceDefinitionV1Beta1Object()
	obj2 = fake.ResourceQuotaObject()

	testSet = []core.Object{obj1, obj2}
)

func TestUpdate(t *testing.T) {
	dr := Resources{}
	objects := testSet
	expectedIDs := getIDs(objects)

	err := dr.Update(objects)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}

	for _, id := range expectedIDs {
		if _, ok := dr.objectSet[id]; !ok {
			t.Errorf("ID %v not found in the declared resource", id)
		}
	}
}

func TestMutateImpossible(t *testing.T) {
	wantResourceVersion := "version 1"

	dr := Resources{}
	o1 := fake.RoleObject(core.Name("foo"), core.Namespace("bar"))
	o1.SetResourceVersion(wantResourceVersion)
	err := dr.Update([]core.Object{o1})
	if err != nil {
		t.Fatal(err)
	}

	o2, found := dr.Get(core.IDOf(o1))
	if !found {
		t.Fatalf("got dr.Get = %v, %t, want dr.Get = obj, true", o2, found)
	}
	o2.SetResourceVersion("version 2")

	o3, found := dr.Get(core.IDOf(o1))
	if !found {
		t.Fatalf("got dr.Get = %v, %t, want dr.Get = obj, true", o2, found)
	}
	if diff := cmp.Diff(wantResourceVersion, o3.GetResourceVersion()); diff != "" {
		t.Error(diff)
	}
}

func asUnstructured(t *testing.T, o runtime.Object) *unstructured.Unstructured {
	t.Helper()
	u, err := reconcile.AsUnstructuredSanitized(o)
	if err != nil {
		t.Fatal("converting to unstructured", err)
	}
	return u
}

func TestDeclarations(t *testing.T) {
	dr := Resources{}
	err := dr.Update(testSet)
	if err != nil {
		t.Fatal(err)
	}

	got := dr.Declarations()
	// Sort got decls to ensure determinism.
	sort.Slice(got, func(i, j int) bool {
		return core.IDOf(got[i]).String() < core.IDOf(got[j]).String()
	})

	want := []*unstructured.Unstructured{
		asUnstructured(t, obj1),
		asUnstructured(t, obj2),
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Error(diff)
	}
}

func TestGet(t *testing.T) {
	dr := Resources{}
	err := dr.Update(testSet)
	if err != nil {
		t.Fatal(err)
	}

	actual, found := dr.Get(core.IDOf(obj1))
	if !found {
		t.Fatal("got not found, want found")
	}
	if diff := cmp.Diff(asUnstructured(t, obj1), actual); diff != "" {
		t.Error(diff)
	}
}

func TestGVKSet(t *testing.T) {
	dr := Resources{}
	err := dr.Update(testSet)
	if err != nil {
		t.Fatal(err)
	}

	got := dr.GVKSet()
	want := map[schema.GroupVersionKind]bool{
		obj1.GroupVersionKind(): true,
		obj2.GroupVersionKind(): true,
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Error(diff)
	}
}

func getIDs(objects []core.Object) []core.ID {
	var IDs []core.ID
	for _, obj := range objects {
		IDs = append(IDs, core.IDOf(obj))
	}
	return IDs
}
