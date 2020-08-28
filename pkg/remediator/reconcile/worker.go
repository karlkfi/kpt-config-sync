package reconcile

import (
	"context"
	"time"

	"github.com/golang/glog"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/declared"
	"github.com/google/nomos/pkg/remediator/queue"
	syncerreconcile "github.com/google/nomos/pkg/syncer/reconcile"
	"k8s.io/apimachinery/pkg/util/wait"
)

// Worker pulls objects from a work queue and passes them to its reconciler for
// remediation.
type Worker struct {
	objectQueue *queue.ObjectQueue
	reconciler  *reconciler
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
}

func (w *Worker) processNextObject(ctx context.Context) bool {
	obj, shutdown := w.objectQueue.Get()
	if shutdown {
		return false
	}

	defer w.objectQueue.Done(obj)
	return w.process(ctx, obj)
}

func (w *Worker) process(ctx context.Context, obj core.Object) bool {
	var toRemediate core.Object
	if queue.WasDeleted(obj) {
		// Passing a nil Object to the reconciler signals that the accompanying ID
		// is for an Object that was deleted.
		toRemediate = nil
	} else {
		toRemediate = obj
	}

	if err := w.reconciler.Remediate(ctx, core.IDOf(obj), toRemediate); err != nil {
		glog.Errorf("Worker received an error while reconciling %q: %v", core.IDOf(obj), err)
		w.objectQueue.AddRateLimited(obj)
		return false
	}

	glog.V(3).Infof("Worker reconciled %q", core.IDOf(obj))
	w.objectQueue.Forget(obj)
	return true
}
