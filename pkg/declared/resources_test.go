package declared

import (
	"context"
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/metrics"
	"github.com/google/nomos/pkg/syncer/reconcile"
	"github.com/google/nomos/pkg/testing/fake"
	"github.com/google/nomos/pkg/testing/testmetrics"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	obj1 = fake.CustomResourceDefinitionV1Beta1Object()
	obj2 = fake.ResourceQuotaObject()

	testSet = []client.Object{obj1, obj2}
	nilSet  = []client.Object{nil}
)

func TestUpdate(t *testing.T) {
	dr := Resources{}
	objects := testSet
	expectedIDs := getIDs(objects)

	err := dr.Update(context.Background(), objects)
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
	o2 := asUnstructured(t, fake.RoleObject(core.Name("baz"), core.Namespace("bar")))
	o2.SetResourceVersion(wantResourceVersion)
	err := dr.Update(context.Background(), []client.Object{o1, o2})
	if err != nil {
		t.Fatal(err)
	}

	// Modify the original resources and ensure the stored resources are preserved.
	o1.SetResourceVersion("version 1++")
	o2.SetResourceVersion("version 1++")

	got1, found := dr.Get(core.IDOf(o1))
	if !found {
		t.Fatalf("got dr.Get = %v, %t, want dr.Get = obj, true", got1, found)
	}
	if diff := cmp.Diff(wantResourceVersion, got1.GetResourceVersion()); diff != "" {
		t.Error(diff)
	}
	got2, found := dr.Get(core.IDOf(o2))
	if !found {
		t.Fatalf("got dr.Get = %v, %t, want dr.Get = obj, true", got2, found)
	}
	if diff := cmp.Diff(wantResourceVersion, got2.GetResourceVersion()); diff != "" {
		t.Error(diff)
	}

	// Modify the fetched resource and ensure the stored resource is preserved.
	got1.SetResourceVersion("version 2")
	got2.SetResourceVersion("version 2")

	got3, found := dr.Get(core.IDOf(o1))
	if !found {
		t.Fatalf("got dr.Get = %v, %t, want dr.Get = obj, true", got3, found)
	}
	if diff := cmp.Diff(wantResourceVersion, got3.GetResourceVersion()); diff != "" {
		t.Error(diff)
	}
	got4, found := dr.Get(core.IDOf(o2))
	if !found {
		t.Fatalf("got dr.Get = %v, %t, want dr.Get = obj, true", got4, found)
	}
	if diff := cmp.Diff(wantResourceVersion, got4.GetResourceVersion()); diff != "" {
		t.Error(diff)
	}
}

func asUnstructured(t *testing.T, o client.Object) *unstructured.Unstructured {
	t.Helper()
	u, err := reconcile.AsUnstructuredSanitized(o)
	if err != nil {
		t.Fatal("converting to unstructured", err)
	}
	return u
}

func TestDeclarations(t *testing.T) {
	dr := Resources{}
	err := dr.Update(context.Background(), testSet)
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
	err := dr.Update(context.Background(), testSet)
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
	err := dr.Update(context.Background(), testSet)
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

func TestResources_InternalErrorMetricValidation(t *testing.T) {
	m := testmetrics.RegisterMetrics(metrics.InternalErrorsView)
	dr := Resources{}
	if err := dr.Update(context.Background(), nilSet); err != nil {
		t.Fatal(err)
	}
	wantMetrics := []*view.Row{
		{Data: &view.CountData{Value: 1}, Tags: []tag.Tag{{Key: metrics.KeyInternalErrorSource, Value: "parser"}}},
	}
	if diff := m.ValidateMetrics(metrics.InternalErrorsView, wantMetrics); diff != "" {
		t.Errorf(diff)
	}
}

func getIDs(objects []client.Object) []core.ID {
	var IDs []core.ID
	for _, obj := range objects {
		IDs = append(IDs, core.IDOf(obj))
	}
	return IDs
}
