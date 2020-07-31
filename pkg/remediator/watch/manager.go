package watch

import (
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/parse/declaredresources"
	"github.com/google/nomos/pkg/remediator/queue"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

// Manager accepts new resource lists that are parsed from Git and then
// updates declared resources and get GVKs.
type Manager struct {
	// cfg is the rest config used to talk to apiserver.
	cfg *rest.Config

	// mapper is the RESTMapper to use for mapping GroupVersionKinds to Resources.
	mapper meta.RESTMapper

	// resources is the declared resources that parsed from Git.
	resources *declaredresources.DeclaredResources

	// queue is the work queue for remediator.
	queue *queue.ObjectQueue

	// watcherMap maps GVKs to their associated watchers
	watcherMap map[schema.GroupVersionKind]Runnable

	// createWatcherFunc is the function to create a watcher.
	createWatcherFunc createWatcherFunc
}

// Options contains options for creating a watch manager.
type Options struct {
	Mapper meta.RESTMapper

	watcherFunc createWatcherFunc
}

// DefaultOptions return the default options:
// - create discovery RESTmapper from the passed rest.Config
// - use createWatcher to create watchers
func DefaultOptions(cfg *rest.Config) (*Options, error) {
	mapper, err := apiutil.NewDiscoveryRESTMapper(cfg)
	if err != nil {
		return nil, err
	}

	return &Options{
		Mapper:      mapper,
		watcherFunc: createWatcher,
	}, nil
}

// NewManager starts a new watch manager
func NewManager(cfg *rest.Config, q *queue.ObjectQueue, options *Options) (*Manager, error) {
	if options == nil {
		var err error
		options, err = DefaultOptions(cfg)
		if err != nil {
			return nil, err
		}
	}

	return &Manager{
		cfg:               cfg,
		resources:         declaredresources.NewDeclaredResources(),
		watcherMap:        make(map[schema.GroupVersionKind]Runnable),
		createWatcherFunc: options.watcherFunc,
		mapper:            options.Mapper,
		queue:             q,
	}, nil
}

// Update accepts new resource list and takes following actions:
// - start watchers for any GroupKind that is present in the new
//   resource list, but weren't present in previous resource list.
// - stop watchers for any GroupVersionKind that is not present
//   in the new resource list
func (m *Manager) Update(objects []core.Object) error {
	err := m.resources.UpdateDecls(objects)
	if err != nil {
		return err
	}

	gvkSet := m.resources.GetGVKSet()

	// Stop obsolete watchers
	for gvk := range m.watcherMap {
		if _, found := gvkSet[gvk]; !found {
			m.stopWatcher(gvk)
		} else {
			// We can remove this GVK from gvkSet because we know
			// it is present in watcherMap so we won't need to start a new watcher for it below.
			delete(gvkSet, gvk)
		}
	}

	// Start new watchers
	for gvk := range gvkSet {
		if _, found := m.watcherMap[gvk]; !found {
			if err := m.startWatcher(gvk); err != nil {
				return err
			}
		}
	}
	return nil
}

// startWatcher starts a watcher for a GVK
func (m *Manager) startWatcher(gvk schema.GroupVersionKind) error {
	_, found := m.watcherMap[gvk]
	if found {
		// The watcher is already started.
		return nil
	}
	opts := watcherOptions{
		gvk:       gvk,
		mapper:    m.mapper,
		config:    m.cfg,
		resources: m.resources,
		queue:     m.queue,
	}
	w, err := m.createWatcherFunc(opts)
	if err != nil {
		return err
	}
	go w.Run()
	m.watcherMap[gvk] = w

	return nil
}

// stopWatcher stops a watcher for a GVK
func (m *Manager) stopWatcher(gvk schema.GroupVersionKind) {
	w, found := m.watcherMap[gvk]
	if !found {
		// The watcher is already stopped.
		return
	}

	// Stop the watcher.
	w.Stop()
	delete(m.watcherMap, gvk)
}
