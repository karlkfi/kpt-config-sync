package watch

import (
	"github.com/google/nomos/pkg/declared"
	"github.com/google/nomos/pkg/remediator/queue"
	"github.com/google/nomos/pkg/status"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
)

type startWatchFunc func(metav1.ListOptions) (watch.Interface, error)

// watcherOptions contains the options needed
// to create a watcher.
type watcherOptions struct {
	gvk        schema.GroupVersionKind
	mapper     meta.RESTMapper
	config     *rest.Config
	resources  *declared.Resources
	queue      *queue.ObjectQueue
	reconciler declared.Scope
	startWatch startWatchFunc
}

// createWatcherFunc is the type of functions to create watchers
type createWatcherFunc func(opts watcherOptions) (Runnable, status.Error)

// createWatcher creates a watcher for a given GVK
func createWatcher(opts watcherOptions) (Runnable, status.Error) {
	if opts.startWatch == nil {
		mapping, err := opts.mapper.RESTMapping(opts.gvk.GroupKind(), opts.gvk.Version)
		if err != nil {
			return nil, status.APIServerErrorf(err, "watcher failed to get REST mapping for %s", opts.gvk.String())
		}

		dynamicClient, err := dynamic.NewForConfig(opts.config)
		if err != nil {
			return nil, status.APIServerErrorf(err, "watcher failed to get dynamic client for %s", opts.gvk.String())
		}

		opts.startWatch = func(options metav1.ListOptions) (watch.Interface, error) {
			return dynamicClient.Resource(mapping.Resource).Watch(options)
		}
	}

	return NewFiltered(opts), nil
}
