package watch

import (
	"github.com/google/nomos/pkg/declared"
	"github.com/google/nomos/pkg/remediator/queue"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
)

// watcherOptions contains the options needed
// to create a watcher.
type watcherOptions struct {
	gvk        schema.GroupVersionKind
	mapper     meta.RESTMapper
	config     *rest.Config
	resources  *declared.Resources
	queue      *queue.ObjectQueue
	reconciler string
}

// createWatcherFunc is the type of functions to create watchers
type createWatcherFunc func(opts watcherOptions) (Runnable, error)

// createWatcher creates a watcher for a given GVK
func createWatcher(opts watcherOptions) (Runnable, error) {
	mapping, err := opts.mapper.RESTMapping(opts.gvk.GroupKind(), opts.gvk.Version)
	if err != nil {
		return nil, err
	}

	dynamicClient, err := dynamic.NewForConfig(opts.config)
	if err != nil {
		return nil, err
	}

	option := metav1.ListOptions{
		Watch: true,
	}
	baseWatcher, err := dynamicClient.Resource(mapping.Resource).Watch(option)
	if err != nil {
		return nil, err
	}

	watcher := &filteredWatcher{
		base:       baseWatcher,
		resources:  opts.resources,
		queue:      opts.queue,
		reconciler: opts.reconciler,
	}

	return watcher, nil
}
