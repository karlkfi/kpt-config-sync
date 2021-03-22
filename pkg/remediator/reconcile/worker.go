package reconcile

import (
	"context"
	"time"

	"github.com/golang/glog"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/declared"
	"github.com/google/nomos/pkg/metrics"
	"github.com/google/nomos/pkg/remediator/queue"
	"github.com/google/nomos/pkg/status"
	syncerclient "github.com/google/nomos/pkg/syncer/client"
	syncerreconcile "github.com/google/nomos/pkg/syncer/reconcile"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Worker pulls objects from a work queue and passes them to its reconciler for
// remediation.
type Worker struct {
	objectQueue queue.Interface
	reconciler  reconcilerInterface
}

// NewWorker returns a new Worker for the given queue and declared resources.
func NewWorker(scope declared.Scope, a syncerreconcile.Applier, q *queue.ObjectQueue, d *declared.Resources) *Worker {
	return &Worker{
		objectQueue: q,
		reconciler:  newReconciler(scope, a, d),
	}
}

// Run starts the Worker pulling objects from its queue for remediation. This
// call blocks until the given context is cancelled.
func (w *Worker) Run(ctx context.Context) {
	wait.UntilWithContext(ctx, func(ctx context.Context) {
		for w.processNextObject(ctx) {
		}
	}, 1*time.Second)
	w.objectQueue.ShutDown()
}

func (w *Worker) processNextObject(ctx context.Context) bool {
	obj, shutdown := w.objectQueue.Get()
	if shutdown {
		return false
	}
	if obj == nil {
		return true
	}

	defer w.objectQueue.Done(obj)
	return w.process(ctx, obj)
}

func (w *Worker) process(ctx context.Context, obj client.Object) bool {
	var toRemediate client.Object
	if queue.WasDeleted(ctx, obj) {
		// Passing a nil Object to the reconciler signals that the accompanying ID
		// is for an Object that was deleted.
		toRemediate = nil
	} else {
		toRemediate = obj
	}

	now := time.Now()
	err := w.reconciler.Remediate(ctx, core.IDOf(obj), toRemediate)
	metrics.RecordRemediateDuration(ctx, metrics.StatusTagKey(err), obj.GetObjectKind().GroupVersionKind(), now)
	if err != nil {
		// To debug the set of events we've missed, you may need to comment out this
		// block. Specifically, this makes things smooth for production, but can
		// hide bugs (for example, if we don't properly process delete events).
		if err.Code() == syncerclient.ResourceConflictCode {
			// This means our cached version of the object isn't the same as the one
			// on the cluster. We need to refresh the cached version.
			metrics.RecordResourceConflict(ctx, obj.GetObjectKind().GroupVersionKind())
			err := w.refresh(ctx, obj)
			if err != nil {
				glog.Errorf("Worker unable to update cached version of %q: %v", core.IDOf(obj), err)
			}
		}

		glog.Errorf("Worker received an error while reconciling %q: %v", core.IDOf(obj), err)
		w.objectQueue.Retry(obj)

		return false
	}

	glog.V(3).Infof("Worker reconciled %q", core.IDOf(obj))
	w.objectQueue.Forget(obj)
	return true
}

// refresh updates the cached version of the object.
func (w *Worker) refresh(ctx context.Context, o client.Object) status.Error {
	c := w.reconciler.GetClient()

	// Try to get an updated version of the object from the cluster.
	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(o.GetObjectKind().GroupVersionKind())
	err := c.Get(ctx, client.ObjectKey{Name: o.GetName(), Namespace: o.GetNamespace()}, u)

	switch {
	case apierrors.IsNotFound(err):
		// The object no longer exists on the cluster, so mark it deleted.
		w.objectQueue.Add(queue.MarkDeleted(ctx, o))
	case err != nil:
		// We encountered some other error that we don't know how to solve, so
		// surface it.
		return status.APIServerError(err, "failed to get updated object for worker cache", o)
	default:
		// Update the cached version of the resource.
		w.objectQueue.Add(u)
	}
	return nil
}
