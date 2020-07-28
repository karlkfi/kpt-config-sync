package watch

import (
	"github.com/golang/glog"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/parse/declaredresources"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/syncer/differ"
	"k8s.io/apimachinery/pkg/watch"
)

// queue is the interface to add an event to the work queue.
type queue interface {
	Add(core.Object)
}

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
	queue     queue
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
		// get core.Object from the runtime object.
		object, err := core.ObjectOf(event.Object)
		if err != nil {
			glog.Warning(status.InternalErrorf("getting core.Object from runtime.Object %v", err))
			continue
		}
		// filter objects.
		if w.ignoreObject(object) {
			glog.V(4).Info("ignore event for unwatched object")
			continue
		}
		w.queue.Add(object)
	}
}

// ignoreObject returns true if the object is not
// managed by Config Sync or not in the declared resources.
func (w *filteredWatcher) ignoreObject(object core.Object) bool {
	// Get id from the runtime object.
	id := core.IDOf(object)
	_, found := w.resources.GetDecl(id)
	return !found && !differ.ManagementEnabled(object)
}
