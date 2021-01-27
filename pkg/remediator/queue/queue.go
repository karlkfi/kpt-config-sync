package queue

import (
	"fmt"
	"sync"

	"github.com/golang/glog"
	"github.com/google/nomos/pkg/core"
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

// Interface is the methods ObjectQueue satisfies.
// See ObjectQueue for method definitions.
type Interface interface {
	Add(obj core.Object)
	Get() (core.Object, bool)
	Done(obj core.Object)
	Forget(obj core.Object)
	Retry(obj core.Object)
	ShutDown()
}

// ObjectQueue is a wrapper around workqueue.RateLimitingInterface for use with
// declared resources. It deduplicates work items by their GVKNN.
type ObjectQueue struct {
	// The workqueue contains work item keys so that it can maintain the order in
	// which those items should be worked on.
	underlying workqueue.RateLimitingInterface
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
func New(name string) *ObjectQueue {
	return &ObjectQueue{
		underlying: workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), name),
		objects:    map[GVKNN]core.Object{},
		dirty:      map[GVKNN]bool{},
	}
}

// Add this to debug the memory leak issue described in http://b/178044278#comment5
// TODO(haiyanmeng): remove this after b/178044278 is fixed.
func (q *ObjectQueue) print() string {
	return fmt.Sprintf("q.objects: %v, q.dirty: %v, len(q.underlying): %v", q.objects, q.dirty, q.underlying.Len())
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
	if obj == nil {
		// Add this to debug the memory leak issue described in http://b/178044278#comment5
		// TODO(haiyanmeng): remove this after b/178044278 is fixed.
		glog.Warningf("Upserting an empty object into queue for %v", gvknn)
	}
	// Add this to debug the memory leak issue described in http://b/178044278#comment5
	// TODO(haiyanmeng): remove this after b/178044278 is fixed.
	glog.Infof("Before adding %v: %v", gvknn, q.print())
	q.objects[gvknn] = obj
	q.dirty[gvknn] = true
	q.underlying.Add(gvknn)
	// Add this to debug the memory leak issue described in http://b/178044278#comment5
	// TODO(haiyanmeng): remove this after b/178044278 is fixed.
	glog.Infof("After adding %v: %v", gvknn, q.print())
}

// Retry schedules the object to be requeued using the rate limiter.
//
// It is possible that the object has been updated since processing began, so we
// don't want to overwrite the changes but we do want to signal that we want
// the object to be processed again.
func (q *ObjectQueue) Retry(obj core.Object) {
	q.objectLock.Lock()
	defer q.objectLock.Unlock()

	gvknn := gvknnOfObject(obj)
	q.dirty[gvknn] = true
	q.underlying.AddRateLimited(gvknn)
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
	item, shutdown := q.underlying.Get()
	if item == nil || shutdown {
		return nil, shutdown
	}

	// Now that we are past that blocking call, we need to lock to prevent
	// concurrent access to our data structures.
	q.objectLock.Lock()
	defer q.objectLock.Unlock()

	gvknn, isID := item.(GVKNN)
	if !isID {
		glog.Warningf("Got non GVKNN from work queue: %v", item)
		q.underlying.Forget(item)
		return nil, false
	}

	obj := q.objects[gvknn]
	if obj == nil {
		// Add this to debug the memory leak issue described in http://b/178044278#comment5
		// TODO(haiyanmeng): remove this after b/178044278 is fixed.
		glog.Warningf("Found no obj for %v", gvknn)
		glog.Info(q.print())
	}
	// TODO(haiyanmeng): remove this after b/178044278 is fixed.
	glog.Infof("Called Get on %v: %p   len(q.underlying): %v", gvknn, obj, q.underlying.Len())
	delete(q.dirty, gvknn)
	glog.V(4).Infof("Fetched object for processing: %v", obj)
	return obj.DeepCopyObject().(core.Object), false
}

// Done marks item as done processing, and if it has been marked as dirty again
// while it was being processed, it will be re-added to the queue for
// re-processing.
func (q *ObjectQueue) Done(obj core.Object) {
	q.objectLock.Lock()
	defer q.objectLock.Unlock()

	gvknn := gvknnOfObject(obj)
	q.underlying.Done(gvknn)

	// Add this to debug the memory leak issue described in http://b/178044278#comment5
	// TODO(haiyanmeng): remove this after b/178044278 is fixed.
	glog.Infof("Called Done on %v: %p   len(q.underlying): %v", gvknn, q.objects[gvknn], q.underlying.Len())
	if q.dirty[gvknn] {
		glog.V(4).Infof("Leaving dirty object reference in place: %v", q.objects[gvknn])
	} else {
		glog.V(2).Infof("Removing clean object reference: %v", q.objects[gvknn])
		delete(q.objects, gvknn)
	}
}

// Forget is a convenience method that allows callers to directly tell the
// RateLimitingInterface to forget a specific core.Object.
func (q *ObjectQueue) Forget(obj core.Object) {
	gvknn := gvknnOfObject(obj)
	q.underlying.Forget(gvknn)
}

// Len returns the length of the underlying queue.
func (q *ObjectQueue) Len() int {
	return q.underlying.Len()
}

// ShutDown shuts down the object queue.
func (q *ObjectQueue) ShutDown() {
	q.underlying.ShutDown()
}
