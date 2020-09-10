package watch

import (
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/declared"
	"github.com/google/nomos/pkg/remediator/queue"
	"github.com/google/nomos/pkg/status"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

// Manager accepts new resource lists that are parsed from Git and then
// updates declared resources and get GVKs.
type Manager struct {
	// reconciler is the name of the reconciler process running the Manager.
	reconciler declared.Scope

	// cfg is the rest config used to talk to apiserver.
	cfg *rest.Config

	// mapper is the RESTMapper to use for mapping GroupVersionKinds to Resources.
	mapper meta.RESTMapper

	// resources is the declared resources that are parsed from Git.
	resources *declared.Resources

	// queue is the work queue for remediator.
	queue *queue.ObjectQueue

	// watcherMap maps GVKs to their associated watchers
	watcherMap map[schema.GroupVersionKind]Runnable

	// createWatcherFunc is the function to create a watcher.
	createWatcherFunc createWatcherFunc
}

// Options contains options for creating a watch manager.
type Options struct {
	// Mapper is the RESTMapper to use for mapping GroupVersionKinds to Resources.
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
func NewManager(reconciler declared.Scope, cfg *rest.Config, q *queue.ObjectQueue, decls *declared.Resources, options *Options) (*Manager, error) {
	if options == nil {
		var err error
		options, err = DefaultOptions(cfg)
		if err != nil {
			return nil, err
		}
	}

	return &Manager{
		reconciler:        reconciler,
		cfg:               cfg,
		resources:         decls,
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
//
// Returns a map from the each declared type to whether we have a watcher for
// that type.
func (m *Manager) Update(objects []core.Object) (map[schema.GroupVersionKind]bool, status.MultiError) {
	err := m.resources.Update(objects)
	if err != nil {
		// This only fails if we've made a coding mistake.
		return nil, err
	}

	watched := m.resources.GVKSet()

	// Stop obsolete watchers.
	for gvk := range m.watcherMap {
		if !watched[gvk] {
			// We were watching the type, but no longer have declarations for it.
			// It is safe to stop the watcher.
			m.stopWatcher(gvk)
		}
	}

	// Start new watchers
	var errs status.MultiError
	for gvk := range watched {
		if _, found := m.watcherMap[gvk]; !found {
			// We don't have a watcher for this type, so add a watcher for it.
			if err := m.startWatcher(gvk); err != nil {
				// We'll try to start the watcher next time - the type may be temporarily
				// unavailable.
				errs = status.Append(errs, err)
				// We aren't watching the type since launching the watcher failed.
				watched[gvk] = false
			}
		}
	}

	return watched, errs
}

// startWatcher starts a watcher for a GVK
func (m *Manager) startWatcher(gvk schema.GroupVersionKind) error {
	_, found := m.watcherMap[gvk]
	if found {
		// The watcher is already started.
		return nil
	}
	opts := watcherOptions{
		gvk:        gvk,
		mapper:     m.mapper,
		config:     m.cfg,
		resources:  m.resources,
		queue:      m.queue,
		reconciler: m.reconciler,
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
