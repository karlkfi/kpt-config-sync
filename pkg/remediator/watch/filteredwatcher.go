package watch

import (
	"math/rand"
	"sync"
	"time"

	"github.com/golang/glog"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/declared"
	"github.com/google/nomos/pkg/diff"
	"github.com/google/nomos/pkg/remediator/queue"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/syncer/differ"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

var (
	// Copying strategy from k8s.io/client-go/tools/cache/reflector.go
	// We try to spread the load on apiserver by setting timeouts for
	// watch requests - it is random in [minWatchTimeout, 2*minWatchTimeout].
	minWatchTimeout = 5 * time.Minute
)

// Runnable defines the custom watch interface.
type Runnable interface {
	Stop()
	Run() status.Error
}

// filteredWatcher is wrapper around a watch interface.
// It only keeps the events for objects that are
// - either present in the declared resources,
// - or managed by Config Sync.
type filteredWatcher struct {
	gvk        string
	startWatch startWatchFunc
	resources  *declared.Resources
	queue      *queue.ObjectQueue
	reconciler declared.Scope

	// The following fields are guarded by the mutex.
	mux     sync.Mutex
	base    watch.Interface
	stopped bool
}

// filteredWatcher implements the Runnable interface.
var _ Runnable = &filteredWatcher{}

// NewFiltered returns a new filtered watch initialized with the given options.
func NewFiltered(cfg watcherConfig) Runnable {
	return &filteredWatcher{
		gvk:        cfg.gvk.String(),
		startWatch: cfg.startWatch,
		resources:  cfg.resources,
		queue:      cfg.queue,
		reconciler: cfg.reconciler,
		base:       watch.NewEmptyWatch(),
	}
}

// Stop fully stops the filteredWatcher in a threadsafe manner. This means that
// it stops the underlying base watch and prevents the filteredWatcher from
// restarting it (like it does if the API server disconnects the base watch).
func (w *filteredWatcher) Stop() {
	w.mux.Lock()
	defer w.mux.Unlock()

	w.base.Stop()
	w.stopped = true
}

// Run reads the event from the base watch interface,
// filters the event and pushes the object contained
// in the event to the controller work queue.
func (w *filteredWatcher) Run() status.Error {
	glog.Infof("Watch started for %s", w.gvk)
	var resourceVersion string

	for {
		// There are three ways this function can return:
		// 1. false, error -> We were unable to start the watch, so exit Run().
		// 2. false, nil   -> We have been stopped via Stop(), so exit Run().
		// 3. true,  nil   -> We have not been stopped and we started a new watch.
		started, err := w.start(resourceVersion)
		if err != nil {
			return err
		}
		if !started {
			break
		}

		glog.V(2).Infof("(Re)starting watch for %s at resource version %q", w.gvk, resourceVersion)
		for event := range w.base.ResultChan() {
			if newVersion := w.handle(event); newVersion != "" {
				resourceVersion = newVersion
			}
		}
		glog.V(2).Infof("Ending watch for %s at resource version %q", w.gvk, resourceVersion)
	}
	glog.Infof("Watch stopped for %s", w.gvk)
	return nil
}

// start initiates a new base watch at the given resource version in a
// threadsafe manner and returns true if the new base watch was created. Returns
// false if the filteredWatcher is already stopped and returns error if the base
// watch could not be started.
func (w *filteredWatcher) start(resourceVersion string) (bool, status.Error) {
	w.mux.Lock()
	defer w.mux.Unlock()

	if w.stopped {
		return false, nil
	}
	w.base.Stop()

	// We want to avoid situations of hanging watchers. Stop any watchers that
	// do not receive any events within the timeout window.
	timeoutSeconds := int64(minWatchTimeout.Seconds() * (rand.Float64() + 1.0))
	options := metav1.ListOptions{
		AllowWatchBookmarks: true,
		ResourceVersion:     resourceVersion,
		TimeoutSeconds:      &timeoutSeconds,
		Watch:               true,
	}

	base, err := w.startWatch(options)
	if err != nil {
		return false, status.APIServerErrorf(err, "failed to start watch for %s", w.gvk)
	}
	w.base = base
	return true, nil
}

// handle reads the event from the base watch interface,
// filters the event and pushes the object contained
// in the event to the controller work queue.
func (w *filteredWatcher) handle(event watch.Event) string {
	var deleted bool
	switch event.Type {
	case watch.Added, watch.Modified:
		deleted = false
	case watch.Deleted:
		deleted = true
	case watch.Bookmark:
		m, err := meta.Accessor(event.Object)
		if err != nil {
			glog.Errorf("Unable to access metadata of Bookmark event: %v", event)
			return ""
		}
		return m.GetResourceVersion()
	default:
		glog.Errorf("Unsupported watch event: %#v", event)
		return ""
	}

	// get core.Object from the runtime object.
	object, err := core.ObjectOf(event.Object)
	if err != nil {
		glog.Warningf("Received non core.Object in watch event: %v", err)
		// TODO(b/162601559): Increment internal error metric here
		return ""
	}
	// filter objects.
	if !w.shouldProcess(object) {
		glog.V(4).Infof("Ignoring event for object: %v", object)
		return object.GetResourceVersion()
	}

	if deleted {
		glog.V(2).Infof("Received watch event for deleted object %q", core.IDOf(object))
		object = queue.MarkDeleted(object)
	} else {
		glog.V(2).Infof("Received watch event for created/updated object %q", core.IDOf(object))
	}

	glog.V(3).Infof("Received object: %v", object)
	w.queue.Add(object)
	return object.GetResourceVersion()
}

// shouldProcess returns true if the given object should be enqueued by the
// watcher for processing.
func (w *filteredWatcher) shouldProcess(object core.Object) bool {
	if !diff.CanManage(w.reconciler, object) {
		return false
	}

	id := core.IDOf(object)
	if decl, ok := w.resources.Get(id); ok {
		// If the object is declared, we only process it if it has the same GVK as
		// its declaration. Otherwise we expect to get another event for the same
		// object but with a matching GVK so we can actually compare it to its
		// declaration.
		return object.GroupVersionKind() == decl.GroupVersionKind()
	}

	// Even if the object is undeclared, we still want to process it if it is
	// tagged as a managed object.
	if !differ.ManagementEnabled(object) {
		return false
	}

	// Only process non declared, managed resources if we are the manager.
	return diff.IsManager(w.reconciler, object)
}
