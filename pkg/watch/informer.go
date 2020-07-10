package watch

import (
	"time"

	"github.com/google/nomos/pkg/parse/declaredresources"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

// informerOptions contains the options needed
// to create an informer.
type informerOptions struct {
	gvk       schema.GroupVersionKind
	mapper    meta.RESTMapper
	config    *rest.Config
	resync    time.Duration
	resources *declaredresources.DeclaredResources
	queue     queue
}

// createInformerFunc is the type of functions to create informers
type createInformerFunc func(opts informerOptions) (mapEntry, error)

// CreateInformer creates a SharedIndexInformer for a given GVK
func createInformer(opts informerOptions) (mapEntry, error) {
	var entry = mapEntry{}
	// create a sharedIndexInformer from a listwatch function
	lw, err := createUnstructuredListWatch(opts.gvk, opts.mapper, opts.config, opts.resources)
	if err != nil {
		return entry, err
	}
	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(opts.gvk)

	informer := cache.NewSharedIndexInformer(lw, obj, opts.resync, cache.Indexers{
		cache.NamespaceIndex: cache.MetaNamespaceIndexFunc,
	})

	// Add an event handler to the informer
	informer.AddEventHandler(handler{resources: opts.resources, queue: opts.queue})

	stopCh := make(chan struct{})
	go func() {
		informer.Run(stopCh)
	}()

	entry.informer = informer
	entry.stopCh = stopCh
	return entry, nil
}
