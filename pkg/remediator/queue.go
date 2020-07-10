package remediator

import (
	"time"

	"github.com/golang/glog"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/watch"
	"k8s.io/client-go/util/workqueue"
)

// GVKNN is the same type as defined in watch.GVKNN
type GVKNN = watch.GVKNN

// queue is a wrapper around workqueue.DelayingInterface for use with declared
// resources. Only deduplicates work items by GVKNN.
type queue struct {
	// work is the queue of work to be done.
	work workqueue.DelayingInterface
}

// newQueue creates a new work queue for use in signalling objects that may need
// remediation.
func newQueue() *queue {
	return &queue{
		work: workqueue.NewDelayingQueue(),
	}
}

// Add marks the item as needing processing.
func (q *queue) Add(id GVKNN) {
	q.AddAfter(id, 0)
}

// AddAfter adds the given item to the work queue after the given delay.
// Blocks until the item has been added.
func (q *queue) AddAfter(id GVKNN, duration time.Duration) {
	if q.work.ShuttingDown() {
		return
	}
	// Blocks until AddAfter adds the item.
	q.work.AddAfter(id, duration)
}

// Len returns the current queue length, for informational purposes only. You
// shouldn't e.g. gate a call to Add() or Get() on Len() being a particular
// value, that can't be synchronized properly.
func (q *queue) Len() int {
	return q.work.Len()
}

// Get blocks until it can return an item to be processed.
//
// Returns the next item to process, and whether the queue has been shut down
// and has no more items to process.
//
// If the queue has been shut down the caller should end their goroutine.
//
// You must call Done with item when you have finished processing it or else the
// item will never be processed again.
func (q *queue) Get() (*GVKNN, bool) {
	item, shutdown := q.work.Get()
	if item == nil || shutdown {
		return nil, shutdown
	}

	gvknn, isID := item.(GVKNN)
	if !isID {
		glog.Warning(status.InternalErrorf("got non GVKNN from work queue: %+v", item))
		return nil, false
	}

	return &gvknn, false
}

// Done marks item as done processing, and if it has been marked as dirty again
// while it was being processed, it will be re-added to the queue for
// re-processing.
func (q *queue) Done(id GVKNN) {
	q.work.Done(id)
}
