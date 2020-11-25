package watch

import (
	"sync"

	"github.com/golang/glog"
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

	// createWatcherFunc is the function to create a watcher.
	createWatcherFunc createWatcherFunc

	// The following fields are guarded by the mutex.
	mux sync.Mutex
	// watcherMap maps GVKs to their associated watchers
	watcherMap map[schema.GroupVersionKind]Runnable
	// needsUpdate indicates if the Manager's watches need to be updated.
	needsUpdate bool
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
	mapper, err := apiutil.NewDynamicRESTMapper(cfg)
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

// NeedsUpdate returns true if the Manager's watches need to be updated. This
// function is threadsafe.
func (m *Manager) NeedsUpdate() bool {
	m.mux.Lock()
	defer m.mux.Unlock()

	return m.needsUpdate
}

// UpdateWatches accepts a map of GVKs that should be watched and takes the
// following actions:
// - stop watchers for any GroupVersionKind that is not present in the given
//   map.
// - start watchers for any GroupVersionKind that is present in the given map
//   and not present in the current watch map.
//
// This function is threadsafe.
func (m *Manager) UpdateWatches(gvkMap map[schema.GroupVersionKind]struct{}) status.MultiError {
	m.mux.Lock()
	defer m.mux.Unlock()

	// Stop obsolete watchers.
	for gvk := range m.watcherMap {
		if _, keepWatching := gvkMap[gvk]; !keepWatching {
			// We were watching the type, but no longer have declarations for it.
			// It is safe to stop the watcher.
			m.stopWatcher(gvk)
		}
	}

	// Start new watchers
	var errs status.MultiError
	for gvk := range gvkMap {
		if _, isWatched := m.watcherMap[gvk]; !isWatched {
			// We don't have a watcher for this type, so add a watcher for it.
			if err := m.startWatcher(gvk); err != nil {
				errs = status.Append(errs, err)
			}
		}
	}

	// If any errors occurred, then the Manager still needs to be updated.
	m.needsUpdate = errs != nil
	return errs
}

// watchedGVKs returns a list of all GroupVersionKinds currently being watched.
func (m *Manager) watchedGVKs() []schema.GroupVersionKind {
	var gvks []schema.GroupVersionKind
	for gvk := range m.watcherMap {
		gvks = append(gvks, gvk)
	}
	return gvks
}

// startWatcher starts a watcher for a GVK. This function is NOT threadsafe;
// caller must have a lock on m.mux.
func (m *Manager) startWatcher(gvk schema.GroupVersionKind) error {
	_, found := m.watcherMap[gvk]
	if found {
		// The watcher is already started.
		return nil
	}
	cfg := watcherConfig{
		gvk:        gvk,
		mapper:     m.mapper,
		config:     m.cfg,
		resources:  m.resources,
		queue:      m.queue,
		reconciler: m.reconciler,
	}
	w, err := m.createWatcherFunc(cfg)
	if err != nil {
		return err
	}

	m.watcherMap[gvk] = w
	go m.runWatcher(w, gvk)
	return nil
}

// runWatcher blocks until the given watcher finishes running. This function is
// threadsafe.
func (m *Manager) runWatcher(r Runnable, gvk schema.GroupVersionKind) {
	if err := r.Run(); err != nil {
		glog.Warningf("Error running watcher for %s: %v", gvk.String(), err)
		m.mux.Lock()
		delete(m.watcherMap, gvk)
		m.needsUpdate = true
		m.mux.Unlock()
	}
}

// stopWatcher stops a watcher for a GVK. This function is NOT threadsafe;
// caller must have a lock on m.mux.
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
