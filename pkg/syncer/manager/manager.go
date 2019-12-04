// Package manager includes controller managers.
package manager

import (
	"context"

	"github.com/golang/glog"
	"github.com/google/nomos/pkg/util/discovery"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// RestartableManager is a controller manager that can be restarted based on the resources it syncs.
type RestartableManager interface {
	// Restart restarts the Manager and all the controllers it manages. The given schema.
	// GroupVersionKinds will be added to the scheme.
	// Returns if a restart actually happened and if there were any errors while doing it.
	Restart(gvks map[schema.GroupVersionKind]bool, scoper discovery.Scoper, force bool) (bool, error)
}

var _ RestartableManager = &SubManager{}

// SubManager is a manager.Manager that is managed by a higher-level controller.
type SubManager struct {
	manager.Manager
	// controllerBuilder builds and initializes controllers for this Manager.
	controllerBuilder ControllerBuilder
	// baseCfg is rest.Config that has no watched resources added to the scheme.
	baseCfg *rest.Config
	// cancel is a cancellation function for ctx. May be nil if ctx is unavailable
	cancel context.CancelFunc
	// errCh gets errors that come up when starting the SubManager
	errCh chan error
}

// NewSubManager returns a new SubManager
func NewSubManager(mgr manager.Manager, controllerBuilder ControllerBuilder, errCh chan error) *SubManager {
	r := &SubManager{
		Manager:           mgr,
		controllerBuilder: controllerBuilder,
		baseCfg:           rest.CopyConfig(mgr.GetConfig()),
		errCh:             errCh,
	}
	return r
}

// context returns a new cancellable context.Context. If a context.Context was previously generated, it cancels it.
func (m *SubManager) context() context.Context {
	if m.cancel != nil {
		m.cancel()
		glog.Info("Stopping SubManager")
	}
	var ctx context.Context
	ctx, m.cancel = context.WithCancel(context.Background())
	return ctx
}

// Restart implements RestartableManager.
func (m *SubManager) Restart(gvks map[schema.GroupVersionKind]bool, scoper discovery.Scoper, force bool) (bool, error) {
	if !force && !m.controllerBuilder.NeedsRestart(gvks) {
		return false, nil
	}
	ctx := m.context()

	var err error
	// MetricsBindAddress 0 disables default metrics for the manager
	// If metrics are enabled, every time the subManger restarts it tries to bind to the metrics port
	// but fails because restarting does not unbind the port.
	// Instead of disabling these metrics, we could figure out a way to unbind the port on restart.
	m.Manager, err = manager.New(rest.CopyConfig(m.baseCfg), manager.Options{MetricsBindAddress: "0"})
	if err != nil {
		return true, errors.Wrap(err, "could not create the Manager for SubManager")
	}

	if err := m.controllerBuilder.UpdateScheme(m.GetScheme(), gvks); err != nil {
		return true, errors.Wrap(err, "could not update the scheme")
	}
	if err := m.controllerBuilder.StartControllers(ctx, m, gvks, scoper); err != nil {
		return true, errors.Wrap(err, "could not start controllers")
	}

	go func() {
		// Propagate errors with starting the SubManager up to the parent controller, so we can restart SubManager.
		m.errCh <- m.Start(ctx.Done())
	}()

	glog.Info("Starting SubManager")
	return true, nil
}
