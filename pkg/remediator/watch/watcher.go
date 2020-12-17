package watch

import (
	"context"

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

// watcherConfig contains the options needed
// to create a watcher.
type watcherConfig struct {
	gvk        schema.GroupVersionKind
	mapper     meta.RESTMapper
	config     *rest.Config
	resources  *declared.Resources
	queue      *queue.ObjectQueue
	reconciler declared.Scope
	startWatch startWatchFunc
}

// createWatcherFunc is the type of functions to create watchers
type createWatcherFunc func(ctx context.Context, cfg watcherConfig) (Runnable, status.Error)

// createWatcher creates a watcher for a given GVK
func createWatcher(ctx context.Context, cfg watcherConfig) (Runnable, status.Error) {
	if cfg.startWatch == nil {
		mapping, err := cfg.mapper.RESTMapping(cfg.gvk.GroupKind(), cfg.gvk.Version)
		if err != nil {
			return nil, status.APIServerErrorf(err, "watcher failed to get REST mapping for %s", cfg.gvk.String())
		}

		dynamicClient, err := dynamic.NewForConfig(cfg.config)
		if err != nil {
			return nil, status.APIServerErrorf(err, "watcher failed to get dynamic client for %s", cfg.gvk.String())
		}

		if cfg.reconciler == declared.RootReconciler {
			cfg.startWatch = func(options metav1.ListOptions) (watch.Interface, error) {
				return dynamicClient.Resource(mapping.Resource).Watch(ctx, options)
			}
		} else {
			cfg.startWatch = func(options metav1.ListOptions) (watch.Interface, error) {
				return dynamicClient.Resource(mapping.Resource).Namespace(string(cfg.reconciler)).Watch(ctx, options)
			}
		}
	}

	return NewFiltered(cfg), nil
}
