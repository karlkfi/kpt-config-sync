package fake

import (
	"context"

	"github.com/google/nomos/pkg/status"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// applier implements a fake reconcile.Applier for use in testing.
//
// reconcile.Applier not imported due to import cycle considerations.
type applier struct {
	*Client
}

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

// Delete implements reconcile.Applier.
func (a *applier) Delete(ctx context.Context, obj *unstructured.Unstructured) (bool, status.Error) {
	err := a.Client.Delete(ctx, obj)
	if err != nil {
		return false, status.APIServerError(err, "deleting")
	}
	return true, nil
}
