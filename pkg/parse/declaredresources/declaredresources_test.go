package declaredresources

import (
	"sort"
	"sync"
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

func TestUpdateDecls(t *testing.T) {
	dr := DeclaredResources{
		mutex: sync.RWMutex{},
	}
	objects := testSet
	expectedIDs := getIDs(objects)

	err := dr.UpdateDecls(objects)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}

	for _, id := range expectedIDs {
		if _, ok := dr.objectSet[id]; !ok {
			t.Errorf("ID %v not found in the declared resource", id)
		}
	}
}

func asUnstructured(t *testing.T, o runtime.Object) *unstructured.Unstructured {
	t.Helper()
	u, err := reconcile.AsUnstructured(o)
	if err != nil {
		t.Fatal("converting to unstructured", err)
	}
	return u
}

func TestGetDecls(t *testing.T) {
	dr := DeclaredResources{}
	err := dr.UpdateDecls(testSet)
	if err != nil {
		t.Fatal(err)
	}

	got := dr.Decls()
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

func TestGetDecl(t *testing.T) {
	dr := DeclaredResources{}
	err := dr.UpdateDecls(testSet)
	if err != nil {
		t.Fatal(err)
	}

	actual, found := dr.GetDecl(core.IDOf(obj1))
	if !found {
		t.Fatal("got not found, want found")
	}
	if diff := cmp.Diff(asUnstructured(t, obj1), actual); diff != "" {
		t.Error(diff)
	}
}

func TestGetGVKSet(t *testing.T) {
	dr := DeclaredResources{}
	err := dr.UpdateDecls(testSet)
	if err != nil {
		t.Fatal(err)
	}

	got := dr.GetGKSet()
	want := map[schema.GroupKind]struct{}{
		obj1.GroupVersionKind().GroupKind(): {},
		obj2.GroupVersionKind().GroupKind(): {},
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
