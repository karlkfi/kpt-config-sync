package remediator

import (
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/parse/declaredresources"
	"github.com/google/nomos/pkg/syncer/reconcile"
)

// Interface updates the declared configuration stored in memory.
type Interface interface {
	UpdateDecls(objects []core.Object) error
}

// Remediator knows how to keep the state of a Kubernetes cluster in sync with
// a set of declared resources. It processes a work queue of items, and ensures
// each matches the set of declarations passed on instantiation.
//
// The exposed Queue operations are threadsafe - multiple callers may safely
// synchronously add and consume work items.
type Remediator struct {
	decls *declaredresources.DeclaredResources
	*queue
}

var _ Interface = &Remediator{}

// New instantiates launches goroutines to make the state of the connected
// cluster match the declared resources.
//
// It is safe for decls to be modified after they have been passed into the
// Remediator.
func New(_ reconcile.Applier, decls *declaredresources.DeclaredResources) *Remediator {
	// TODO(b/157587458): Launch goroutine(s) that do remediation.
	return &Remediator{
		decls: decls,
		queue: newQueue(),
	}
}

// UpdateDecls implements Interface.
func (r *Remediator) UpdateDecls(objects []core.Object) error {
	return r.decls.UpdateDecls(objects)
}

// NoOp is a Remediator that takes no actions whatsoever.
type NoOp struct{}

// UpdateDecls implements Interface.
func (r *NoOp) UpdateDecls(_ []core.Object) error {
	return nil
}
