package watch

import (
	"github.com/golang/glog"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/parse/declaredresources"
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
	base      watch.Interface
	resources *declaredresources.DeclaredResources
	queue     *queue.ObjectQueue
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
		if w.ignoreObject(object) {
			glog.V(4).Infof("Ignoring event for unmanaged object: %v", object)
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

// ignoreObject returns true if the object is not managed by Config Sync or not
// in the declared resources.
func (w *filteredWatcher) ignoreObject(object core.Object) bool {
	id := core.IDOf(object)
	if declared, ok := w.resources.GetDecl(id); ok {
		// If the object is declared, we only ignore it if it has a different GVK
		// than the declaration. In that case we expect to get another event for the
		// same object but with a matching GVK so we can actually compare it to its
		// declaration.
		return object.GroupVersionKind().String() != declared.GroupVersionKind().String()
	}
	return !differ.ManagementEnabled(object)
}
