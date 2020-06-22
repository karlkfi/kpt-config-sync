package declaredresources

import (
	"reflect"
	"sync"
	"testing"

	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/testing/fake"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	obj1    = fake.CustomResourceDefinitionV1Beta1()
	obj2    = fake.ResourceQuota()
	testSet = map[core.ID]*ast.FileObject{
		core.IDOf(obj1): &obj1,
		core.IDOf(obj2): &obj2,
	}
)

func TestUpdateDecls(t *testing.T) {
	dr := DeclaredResources{
		mutex: sync.RWMutex{},
	}
	objects := []ast.FileObject{obj1, obj2}
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

func TestGetDecls(t *testing.T) {
	dr := DeclaredResources{
		objectSet: testSet,
	}
	actual := dr.Decls()
	if !reflect.DeepEqual(actual, []*ast.FileObject{&obj1, &obj2}) &&
		!reflect.DeepEqual(actual, []*ast.FileObject{&obj2, &obj1}) {
		t.Errorf("actual declared resources isn't as expected")
	}
}

func TestGetDecl(t *testing.T) {
	dr := DeclaredResources{
		objectSet: testSet,
	}

	actual, err := dr.GetDecl(fake.CustomResourceDefinitionV1Beta1Object())
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	if !reflect.DeepEqual(actual, &obj1) {
		t.Errorf("expected %v\nbut got %v", obj1, actual)
	}
}

func TestGetGVKSet(t *testing.T) {
	dr := DeclaredResources{
		objectSet: testSet,
	}
	gvkSet := dr.GetGKSet()
	expected := map[schema.GroupKind]struct{}{
		obj1.GroupVersionKind().GroupKind(): {},
		obj2.GroupVersionKind().GroupKind(): {},
	}
	if !reflect.DeepEqual(gvkSet, expected) {
		t.Errorf("expected %v\nbut got %v", expected, gvkSet)
	}
}

func getIDs(objects []ast.FileObject) []core.ID {
	var IDs []core.ID
	for _, obj := range objects {
		IDs = append(IDs, core.IDOf(obj))
	}
	return IDs
}
