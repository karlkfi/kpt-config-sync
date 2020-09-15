package fake

import (
	"context"

	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/syncer/reconcile"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// applier implements a fake reconcile.Applier for use in testing.
//
// reconcile.Applier not imported due to import cycle considerations.
type applier struct {
	*Client
}

var _ reconcile.Applier = &applier{}

// Create implements reconcile.Applier.
func (a *applier) Create(ctx context.Context, obj *unstructured.Unstructured) (bool, status.Error) {
	err := a.Client.Create(ctx, obj)
	if err != nil {
		return false, status.APIServerError(err, "creating")
	}
	return true, nil
}

// Update implements reconcile.Applier.
func (a *applier) Update(ctx context.Context, intendedState, currentState *unstructured.Unstructured) (bool, status.Error) {
	err := a.Client.Update(ctx, intendedState)
	if err != nil {
		return false, status.APIServerError(err, "updating")
	}
	return true, nil
}

// RemoveNomosMeta implements reconcile.Applier.
func (a *applier) RemoveNomosMeta(ctx context.Context, intent *unstructured.Unstructured) (bool, status.Error) {
	updated := reconcile.RemoveNomosLabelsAndAnnotations(intent)
	if !updated {
		return false, nil
	}

	err := a.Client.Update(ctx, intent)
	if err != nil {
		return false, status.APIServerError(err, "removing meta")
	}
	return true, nil
}

// Delete implements reconcile.Applier.
func (a *applier) Delete(ctx context.Context, obj *unstructured.Unstructured) (bool, status.Error) {
	err := a.Client.Delete(ctx, obj)
	if err != nil {
		return false, status.APIServerError(err, "deleting")
	}
	return true, nil
}

// GetClient implements reconcile.Applier.
func (a *applier) GetClient() client.Client {
	return a.Client
}
