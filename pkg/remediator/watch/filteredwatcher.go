package watch

import (
	"github.com/golang/glog"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/declared"
	"github.com/google/nomos/pkg/diff"
	"github.com/google/nomos/pkg/remediator/queue"
	"github.com/google/nomos/pkg/syncer/differ"
	"k8s.io/apimachinery/pkg/watch"
)

// Runnable defines the custom watch interface.
type Runnable interface {
	Stop()
	Run()
}

// filteredWatcher is wrapper around a watch interface.
// It only keeps the events for objects that are
// - either present in the declared resources,
// - or managed by Config Sync.
type filteredWatcher struct {
	base       watch.Interface
	resources  *declared.Resources
	queue      *queue.ObjectQueue
	reconciler declared.Scope
}

// filteredWatcher implements the Runnable interface.
var _ Runnable = &filteredWatcher{}

// Stop delegates to base watcher.Stop.
func (w *filteredWatcher) Stop() { w.base.Stop() }

// Run reads the event from the base watch interface,
// filters the event and pushes the object contained
// in the event to the controller work queue.
func (w *filteredWatcher) Run() {
	for event := range w.base.ResultChan() {
		var deleted bool
		switch event.Type {
		case watch.Added, watch.Modified:
			deleted = false
		case watch.Deleted:
			deleted = true
		case watch.Bookmark:
			glog.V(4).Infof("Ignoring Bookmark watch event: %#v", event)
			continue
		default:
			glog.Errorf("Unsupported watch event: %#v", event)
			continue
		}

		// get core.Object from the runtime object.
		object, err := core.ObjectOf(event.Object)
		if err != nil {
			glog.Warningf("Received non core.Object in watch event: %v", err)
			// TODO(b/162601559): Increment internal error metric here
			continue
		}
		// filter objects.
		if !w.shouldProcess(object) {
			glog.V(4).Infof("Ignoring event for object: %v", object)
			continue
		}

		if deleted {
			glog.V(2).Infof("Received watch event for deleted object %q", core.IDOf(object))
			object = queue.MarkDeleted(object)
		} else {
			glog.V(2).Infof("Received watch event for created/updated object %q", core.IDOf(object))
		}

		glog.V(3).Infof("Received object: %v", object)
		w.queue.Add(object)
	}
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
		return object.GroupVersionKind().String() == decl.GroupVersionKind().String()
	}
	// Even if the object is undeclared, we still want to process it if it is
	// tagged as a managed object.
	return differ.ManagementEnabled(object)
}
