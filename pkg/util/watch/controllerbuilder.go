package watch

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// ControllerBuilder builds controllers. It is managed by RestartableManager, which is managed by a higher-level controller.
type ControllerBuilder interface {
	// StartControllers starts the relevant controllers using the RestartableManager to manage them.
	StartControllers(ctx context.Context, mgr manager.Manager, gvks map[schema.GroupVersionKind]bool) error
}
