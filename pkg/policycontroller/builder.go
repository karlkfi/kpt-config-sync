package policycontroller

import (
	"context"

	"github.com/google/nomos/pkg/policycontroller/constraint"
	"github.com/google/nomos/pkg/util/watch"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type builder struct{}

var _ watch.ControllerBuilder = &builder{}

// StartControllers starts a new constraint controller for each of the specified
// constraint GVKs.
func (b *builder) StartControllers(ctx context.Context, mgr manager.Manager, gvks map[schema.GroupVersionKind]bool) error {
	for gvk := range gvks {
		if err := constraint.AddController(ctx, mgr, gvk.Kind); err != nil {
			return errors.Wrapf(err, "controller for %s", gvk.String())
		}
	}
	return nil
}
