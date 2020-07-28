package queue

import (
	"github.com/golang/glog"
	"github.com/google/nomos/pkg/core"
)

// ObjectQueue provides type-safe convenience functions for the underlying
// objectQueue.
type ObjectQueue struct {
	*objectQueue
}

// NewNamed creates a new ObjectQueue for remediating resources.
func NewNamed(name string) *ObjectQueue {
	return &ObjectQueue{
		named(name),
	}
}

// Add marks item as needing processing.
func (q *ObjectQueue) Add(obj core.Object) {
	q.objectQueue.Add(obj)
}

// Get blocks until it can return an item to be processed. If shutdown = true,
// the caller should end their goroutine. You must call Done with item when you
// have finished processing it.
func (q *ObjectQueue) Get() (core.Object, bool) {
	item, shutdown := q.objectQueue.Get()
	obj, ok := item.(core.Object)
	if !ok {
		glog.Errorf("Get() received non-core.Object item: %v", item)
	}
	return obj, shutdown
}

// Done marks item as done processing, and if it has been marked as dirty again
// while it was being processed, it will be re-added to the queue for
// re-processing.
func (q *ObjectQueue) Done(obj core.Object) {
	q.objectQueue.Done(obj)
}
