package remediator

import (
	"context"

	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/parse/declaredresources"
	"github.com/google/nomos/pkg/remediator/queue"
	"github.com/google/nomos/pkg/remediator/reconcile"
	syncerreconcile "github.com/google/nomos/pkg/syncer/reconcile"
)

// Remediator knows how to keep the state of a Kubernetes cluster in sync with
// a set of declared resources. It processes a work queue of items, and ensures
// each matches the set of declarations passed on instantiation.
//
// The exposed Queue operations are threadsafe - multiple callers may safely
// synchronously add and consume work items.
type Remediator struct {
	decls       *declaredresources.DeclaredResources
	objectQueue *queue.ObjectQueue
	workers     []*reconcile.Worker
	started     bool
}

// New instantiates launches goroutines to make the state of the connected
// cluster match the declared resources.
//
// It is safe for decls to be modified after they have been passed into the
// Remediator.
func New(name string, applier syncerreconcile.Applier, decls *declaredresources.DeclaredResources, numWorkers int) *Remediator {
	// TODO(b/157587458): Integrate watch manager here and in UpdateDecls()

	q := queue.NewNamed(name)
	workers := make([]*reconcile.Worker, numWorkers)
	for i := 0; i < numWorkers; i++ {
		workers[i] = reconcile.NewWorker(applier, q, decls)
	}

	return &Remediator{
		decls:       decls,
		objectQueue: q,
		workers:     workers,
	}
}

// Start implements Interface
func (r *Remediator) Start(ctx context.Context) {
	if r.started {
		return
	}
	for _, worker := range r.workers {
		go worker.Run(ctx)
	}
	r.started = true
}

// UpdateDecls implements Interface.
func (r *Remediator) UpdateDecls(objects []core.Object) error {
	return r.decls.UpdateDecls(objects)
}
