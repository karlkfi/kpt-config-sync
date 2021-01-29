package queue

import (
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

// ObjectQueue is a wrapper around a workqueue.Interface for use with declared
// resources. It deduplicates work items by their GVKNN.
// NOTE: This was originally designed to wrap a DelayingInterface, but we have
// had to copy a lot of that logic here. At some point it may make sense to
// remove the underlying workqueue.Interface and just consolidate copied logic
// here.
type ObjectQueue struct {
	// cond is a locking condition which allows us to lock all mutating calls but
	// also allow any call to yield the lock safely (specifically for Get).
	cond *sync.Cond
	// rateLimiter enables the ObjectQueue to support rate-limited retries.
	rateLimiter workqueue.RateLimiter
	// delayer is a wrapper around the ObjectQueue which supports delayed Adds.
	delayer workqueue.DelayingInterface
	// underlying is the workqueue that contains work item keys so that it can
	// maintain the order in which those items should be worked on.
	underlying workqueue.Interface
	// objects is a map of actual work items which need to be processed.
	objects map[GVKNN]core.Object
	// dirty is a map of object keys which will need to be reprocessed even if
	// they are currently being processed. This is explained further in Add().
	dirty map[GVKNN]bool
}

// New creates a new work queue for use in signalling objects that may need
// remediation.
func New(name string) *ObjectQueue {
	oq := &ObjectQueue{
		cond:        sync.NewCond(&sync.Mutex{}),
		rateLimiter: workqueue.DefaultControllerRateLimiter(),
		underlying:  workqueue.NewNamed(name),
		objects:     map[GVKNN]core.Object{},
		dirty:       map[GVKNN]bool{},
	}
	oq.delayer = delayingWrap(oq, name)
	return oq
}

// Add marks the object as needing processing unless the object is already in
// the queue AND the existing object is more current than the new one.
func (q *ObjectQueue) Add(obj core.Object) {
	q.cond.L.Lock()
	defer q.cond.L.Unlock()

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
	q.underlying.Add(gvknn)

	if !q.dirty[gvknn] {
		q.dirty[gvknn] = true
		q.cond.Signal()
	}
}

// Retry schedules the object to be requeued using the rate limiter.
func (q *ObjectQueue) Retry(obj core.Object) {
	gvknn := gvknnOfObject(obj)
	q.delayer.AddAfter(obj, q.rateLimiter.When(gvknn))
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
	q.cond.L.Lock()
	defer q.cond.L.Unlock()

	// This is a yielding block that will allow Add() and Done() to be called
	// while it blocks.
	for q.underlying.Len() == 0 && !q.underlying.ShuttingDown() {
		q.cond.Wait()
	}

	item, shutdown := q.underlying.Get()
	if item == nil || shutdown {
		return nil, shutdown
	}

	gvknn, isID := item.(GVKNN)
	if !isID {
		glog.Warningf("Got non GVKNN from work queue: %v", item)
		q.underlying.Done(item)
		q.rateLimiter.Forget(item)
		return nil, false
	}

	obj := q.objects[gvknn]
	delete(q.dirty, gvknn)
	glog.V(4).Infof("Fetched object for processing: %v", obj)
	return obj.DeepCopyObject().(core.Object), false
}

// Done marks item as done processing, and if it has been marked as dirty again
// while it was being processed, it will be re-added to the queue for
// re-processing.
func (q *ObjectQueue) Done(obj core.Object) {
	q.cond.L.Lock()
	defer q.cond.L.Unlock()

	gvknn := gvknnOfObject(obj)
	q.underlying.Done(gvknn)

	if q.dirty[gvknn] {
		glog.V(4).Infof("Leaving dirty object reference in place: %v", q.objects[gvknn])
		q.cond.Signal()
	} else {
		glog.V(2).Infof("Removing clean object reference: %v", q.objects[gvknn])
		delete(q.objects, gvknn)
	}
}

// Forget is a convenience method that allows callers to directly tell the
// RateLimitingInterface to forget a specific core.Object.
func (q *ObjectQueue) Forget(obj core.Object) {
	gvknn := gvknnOfObject(obj)
	q.rateLimiter.Forget(gvknn)
}

// Len returns the length of the underlying queue.
func (q *ObjectQueue) Len() int {
	return q.underlying.Len()
}

// ShutDown shuts down the object queue.
func (q *ObjectQueue) ShutDown() {
	q.underlying.ShutDown()
}

// ShuttingDown returns true if the object queue is shutting down.
func (q *ObjectQueue) ShuttingDown() bool {
	return q.underlying.ShuttingDown()
}

// delayingWrap returns the given ObjectQueue wrapped in a DelayingInterface to
// enable rate-limited retries.
func delayingWrap(oq *ObjectQueue, name string) workqueue.DelayingInterface {
	gw := &genericWrapper{oq}
	return workqueue.NewDelayingQueueWithCustomQueue(gw, name)
}

// genericWrapper is an internal wrapper that allows us to pass an ObjectQueue
// to a DelayingInterface to enable rate-limited retries. It uses unsafe type
// conversion because it should only ever be used in this file and so we know
// that all of the item interfaces are actually core.Objects.
type genericWrapper struct {
	oq *ObjectQueue
}

var _ workqueue.Interface = &genericWrapper{}

// Add implements workqueue.Interface.
func (g *genericWrapper) Add(item interface{}) {
	g.oq.Add(item.(core.Object))
}

// Get implements workqueue.Interface.
func (g *genericWrapper) Get() (item interface{}, shutdown bool) {
	return g.oq.Get()
}

// Done implements workqueue.Interface.
func (g *genericWrapper) Done(item interface{}) {
	g.oq.Done(item.(core.Object))
}

// Len implements workqueue.Interface.
func (g *genericWrapper) Len() int {
	return g.oq.Len()
}

// ShutDown implements workqueue.Interface.
func (g *genericWrapper) ShutDown() {
	g.oq.ShutDown()
}

// ShuttingDown implements workqueue.Interface.
func (g *genericWrapper) ShuttingDown() bool {
	return g.oq.ShuttingDown()
}
