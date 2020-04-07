package scheme

import (
	"reflect"

	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/util/discovery"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// AddToSchemeAsUnstructured adds the GroupVersionKinds to the scheme as unstructured.Unstructured objects.
func AddToSchemeAsUnstructured(scheme *runtime.Scheme, gvks map[schema.GroupVersionKind]bool) {
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
func resourceTypes(
	gvks map[schema.GroupVersionKind]bool,
	scheme *runtime.Scheme,
) (map[schema.GroupVersionKind]runtime.Object, error) {
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

// ResourceScopes returns two slices representing the namespace and cluster scoped resource types with sync enabled.
func ResourceScopes(
	gvks map[schema.GroupVersionKind]bool,
	scheme *runtime.Scheme,
	scoper discovery.Scoper,
) (map[schema.GroupVersionKind]runtime.Object, map[schema.GroupVersionKind]runtime.Object, error) {
	rts, err := resourceTypes(gvks, scheme)
	if err != nil {
		return nil, nil, err
	}
	namespace := make(map[schema.GroupVersionKind]runtime.Object)
	cluster := make(map[schema.GroupVersionKind]runtime.Object)
	for gvk, obj := range rts {
		if gvk.GroupKind() == kinds.CustomResourceDefinitionV1Beta1().GroupKind() {
			// CRDs are handled in the CRD controller and shouldn't be handled in any of SubManager's controllers.
			continue
		}

		isNamespaced, err := scoper.GetGroupKindScope(gvk.GroupKind())
		if err != nil {
			return nil, nil, err
		}

		if isNamespaced {
			namespace[gvk] = obj
		} else {
			cluster[gvk] = obj
		}
	}
	return namespace, cluster, nil
}
