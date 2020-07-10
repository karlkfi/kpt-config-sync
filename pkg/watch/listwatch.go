package watch

import (
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"

	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/parse/declaredresources"
)

func createUnstructuredListWatch(gvk schema.GroupVersionKind, mapper meta.RESTMapper, config *rest.Config,
	resources *declaredresources.DeclaredResources) (*cache.ListWatch, error) {
	// Kubernetes APIs work against Resources, not GroupVersionKinds.  Map the
	// groupVersionKind to the Resource API we will use.
	mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return nil, err
	}
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	// Create a new ListWatch for the obj
	return &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			listObj, err := dynamicClient.Resource(mapping.Resource).List(opts)
			if err != nil {
				return nil, err
			}
			return filterObject(listObj, resources), nil
		},
		// Setup the watch function
		// TODO: The watcher should filter events and only keep the ones
		// can be found in the declared resources.
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			// Watch needs to be set to true separately
			opts.Watch = true
			return dynamicClient.Resource(mapping.Resource).Watch(opts)
		},
	}, nil
}

// filterObject only keeps the object that is
// - either present in the declared resources
// - or managed by Config Sync
func filterObject(ul *unstructured.UnstructuredList, resources *declaredresources.DeclaredResources) *unstructured.UnstructuredList {
	newItems := []unstructured.Unstructured{}
	for _, u := range ul.Items {
		id := core.IDOfUnstructured(u)
		_, found := resources.GetDecl(id)
		if found || managementEnabled(&u) {
			newItems = append(newItems, u)
		}
	}
	ul.Items = newItems
	return ul
}
