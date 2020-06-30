package watch

import (
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

func createUnstructuredListWatch(gvk schema.GroupVersionKind, mapper meta.RESTMapper, config *rest.Config) (*cache.ListWatch, error) {
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
			return dynamicClient.Resource(mapping.Resource).List(opts)
		},
		// Setup the watch function
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			// Watch needs to be set to true separately
			opts.Watch = true
			return dynamicClient.Resource(mapping.Resource).Watch(opts)
		},
	}, nil
}

// createInformerFunc is the type of functions to create informers
type createInformerFunc func(schema.GroupVersionKind, meta.RESTMapper, *rest.Config,
	time.Duration) (mapEntry, error)

// CreateInformer creates a SharedIndexInformer for a given GVK
func createInformer(gvk schema.GroupVersionKind, mapper meta.RESTMapper, config *rest.Config,
	resync time.Duration) (mapEntry, error) {
	var entry = mapEntry{}
	// create a sharedIndexInformer from a listwatch function
	lw, err := createUnstructuredListWatch(gvk, mapper, config)
	if err != nil {
		return entry, err
	}
	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(gvk)

	informer := cache.NewSharedIndexInformer(lw, obj, resync, cache.Indexers{
		cache.NamespaceIndex: cache.MetaNamespaceIndexFunc,
	})

	stopCh := make(chan struct{})
	go func() {
		informer.Run(stopCh)
	}()

	entry.informer = informer
	entry.stopCh = stopCh
	return entry, nil
}
