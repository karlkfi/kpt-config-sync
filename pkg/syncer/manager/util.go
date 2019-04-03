package manager

import (
	"reflect"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// addtoSchemeUnstructured adds the GroupVersionKinds to the scheme as unstructured.Unstructured objects.
func addToSchemeUnstructured(scheme *runtime.Scheme, gvks map[schema.GroupVersionKind]bool) {
	for gvk := range gvks {
		if !scheme.Recognizes(gvk) {
			scheme.AddKnownTypeWithName(gvk, &unstructured.Unstructured{})
			gvkList := schema.GroupVersionKind{
				Group:   gvk.Group,
				Version: gvk.Version,
				Kind:    gvk.Kind + "List",
			}
			scheme.AddKnownTypeWithName(gvkList, &unstructured.UnstructuredList{})
			metav1.AddToGroupVersion(scheme, gvk.GroupVersion())
		}
	}
}

// resourceTypes returns all the sync enabled resources and the corresponding type stored in the scheme.
func resourceTypes(gvks map[schema.GroupVersionKind]bool,
	scheme *runtime.Scheme) (map[schema.GroupVersionKind]runtime.Object, error) {
	knownGVKs := scheme.AllKnownTypes()
	m := make(map[schema.GroupVersionKind]runtime.Object)
	for gvk := range gvks {
		rt, ok := knownGVKs[gvk]
		if !ok {
			return nil, errors.Errorf("trying to sync %q, which hasn't been registered in the scheme", gvk)
		}

		// If it's a resource with an unknown type at compile time, we need to specifically set the GroupVersionKind for it
		// when enabling the watch.
		if rt.AssignableTo(reflect.TypeOf(unstructured.Unstructured{})) {
			u := &unstructured.Unstructured{}
			u.SetGroupVersionKind(gvk)
			m[gvk] = u
		} else {
			m[gvk] = reflect.New(rt).Interface().(runtime.Object)
		}
	}
	return m, nil
}
