package remediator

import (
	"context"

	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/declared"
	"github.com/google/nomos/pkg/remediator/queue"
	"github.com/google/nomos/pkg/remediator/reconcile"
	"github.com/google/nomos/pkg/remediator/watch"
	syncerreconcile "github.com/google/nomos/pkg/syncer/reconcile"
	"github.com/pkg/errors"
	"k8s.io/client-go/rest"
)

// Remediator knows how to keep the state of a Kubernetes cluster in sync with
// a set of declared resources. It processes a work queue of items, and ensures
// each matches the set of declarations passed on instantiation.
//
// The exposed Queue operations are threadsafe - multiple callers may safely
// synchronously add and consume work items.
type Remediator struct {
	watchMgr *watch.Manager
	workers  []*reconcile.Worker
	started  bool
}

// Interface is a fake-able subset of the interface Remediator implements that
// accepts a new set of declared configuration.
//
// Placed here to make discovering the production implementation (above) easier.
type Interface interface {
	Update(objects []core.Object) error
}

var _ Interface = &Remediator{}

// New instantiates launches goroutines to make the state of the connected
// cluster match the declared resources.
//
// It is safe for decls to be modified after they have been passed into the
// Remediator.
func New(reconciler declared.Scope, cfg *rest.Config, applier syncerreconcile.Applier, decls *declared.Resources, numWorkers int) (*Remediator, error) {
	q := queue.NewNamed(string(reconciler))
	workers := make([]*reconcile.Worker, numWorkers)
	for i := 0; i < numWorkers; i++ {
		workers[i] = reconcile.NewWorker(reconciler, applier, q, decls)
	}

	watchMgr, err := watch.NewManager(reconciler, cfg, q, decls, nil)
	if err != nil {
		return nil, errors.Wrap(err, "creating watch manager")
	}

	return &Remediator{
		watchMgr: watchMgr,
		workers:  workers,
	}, nil
}

// Start begins the asynchronous processes for the Remediator's reconcile workers.
func (r *Remediator) Start(ctx context.Context) {
	if r.started {
		return
	}
	for _, worker := range r.workers {
		go worker.Run(ctx)
	}
	r.started = true
}

// Update updates the declared resources for all reconcile workers and
// potentially starts/stops server-side watches.
func (r *Remediator) Update(objects []core.Object) error {
	return r.watchMgr.Update(objects)
}
