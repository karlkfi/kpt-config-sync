package queue

import (
	"sync"

	"github.com/golang/glog"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/status"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/util/workqueue"
)

// GVKNN adds Version to core.ID to make it suitable for getting an object from
// a cluster into an *unstructured.Unstructured.
type GVKNN struct {
	core.ID
	Version string
}

// GroupVersionKind returns the GVK contained in this GVKNN.
func (gvknn GVKNN) GroupVersionKind() schema.GroupVersionKind {
	return gvknn.GroupKind.WithVersion(gvknn.Version)
}

func gvknnOfObject(obj core.Object) GVKNN {
	return GVKNN{
		ID:      core.IDOf(obj),
		Version: obj.GroupVersionKind().Version,
	}
}

// ObjectQueue is a wrapper around workqueue.DelayingInterface for use with
// declared resources. It deduplicates work items by their GVKNN.
type ObjectQueue struct {
	// The workqueue contains work item keys so that it can maintain the order in
	// which those items should be worked on.
	workqueue.DelayingInterface
	// objects is a map of actual work items which need to be processed.
	objects map[GVKNN]core.Object
	// dirty is a map of object keys which will need to be reprocessed even if
	// they are currently being processed. This is explained further in Add().
	dirty map[GVKNN]bool
	// objectLock prevents concurrent access to `objects` and `dirty`.
	objectLock sync.Mutex
}

// New creates a new work queue for use in signalling objects that may need
// remediation.
func New() *ObjectQueue {
	return &ObjectQueue{
		DelayingInterface: workqueue.NewDelayingQueue(),
		objects:           map[GVKNN]core.Object{},
		dirty:             map[GVKNN]bool{},
	}
}

// Add marks the object as needing processing unless the object is already in
// the queue AND the existing object is more current than the new one.
func (q *ObjectQueue) Add(obj core.Object) {
	q.objectLock.Lock()
	defer q.objectLock.Unlock()

	gvknn := gvknnOfObject(obj)

	// Generation is not incremented when metadata is changed. Therefore if
	// generation is equal, we default to accepting the new object as it may have
	// new labels or annotations or other metadata.
	if current, ok := q.objects[gvknn]; ok && current.GetGeneration() > obj.GetGeneration() {
		glog.V(4).Infof("Queue already contains object %q with generation %d; ignoring object: %v", gvknn, current.GetGeneration(), obj)
		return
	}

	// It is possible that a reconciler has already pulled the object for this
	// GVKNN out of the queue and is actively processing it. In that case, we
	// need to mark it dirty here so that it gets re-processed. Eg:
	// 1. q.objects contains generation 1 of a resource.
	// 2. A reconciler pulls gen1 out of the queue to process.
	// 3. The gvknn is no longer marked dirty (see Get() below).
	// 3. Another process/user updates the resource in parallel.
	// 4. The API server notifies the watcher which calls Add() with gen2 of the resource.
	// 5. We insert gen2 and re-mark the gvknn as dirty.
	// 6. The reconciler finishes processing gen1 of the resource and calls Done().
	// 7. Since the gvknn is still marked dirty, we leave the resource in q.objects.
	// 8. Eventually a reconciler pulls gen2 of the resource out of the queue for processing.
	// 9. The gvknn is no longer marked dirty.
	// 10. The reconciler finishes processing gen2 of the resource and calls Done().
	// 11. Since the gvknn is not marked dirty, we remove the resource from q.objects.
	glog.V(2).Infof("Upserting object into queue: %v", obj)
	q.objects[gvknn] = obj
	q.dirty[gvknn] = true
	q.DelayingInterface.Add(gvknn)
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
func (q *ObjectQueue) Get() (core.Object, bool) {
	// This call is a yielding block that will allow Add() and Done() to be called
	// while it blocks.
	item, shutdown := q.DelayingInterface.Get()
	if item == nil || shutdown {
		return nil, shutdown
	}

	// Now that we are past that blocking call, we need to lock to prevent
	// concurrent access to our data structures.
	q.objectLock.Lock()
	defer q.objectLock.Unlock()

	gvknn, isID := item.(GVKNN)
	if !isID {
		glog.Warning(status.InternalErrorf("got non GVKNN from work queue: %+v", item))
		return nil, false
	}

	obj := q.objects[gvknn]
	delete(q.dirty, gvknn)
	glog.V(4).Infof("Fetched object for processing: %v", obj)
	return obj, false
}

// Done marks item as done processing, and if it has been marked as dirty again
// while it was being processed, it will be re-added to the queue for
// re-processing.
func (q *ObjectQueue) Done(obj core.Object) {
	q.objectLock.Lock()
	defer q.objectLock.Unlock()

	gvknn := gvknnOfObject(obj)
	q.DelayingInterface.Done(gvknn)

	if q.dirty[gvknn] {
		glog.V(4).Infof("Leaving dirty object reference in place: %v", q.objects[gvknn])
	} else {
		glog.V(2).Infof("Removing clean object reference: %v", q.objects[gvknn])
		delete(q.objects, gvknn)
	}
}
