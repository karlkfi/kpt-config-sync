/*
Copyright 2018 The CSP Config Management Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

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
	Restart(gvks map[schema.GroupVersionKind]bool, apirs *discovery.APIInfo, force bool) error
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
func NewSubManager(mgr manager.Manager, cfg *rest.Config, controllerBuilder ControllerBuilder, errCh chan error) *SubManager {
	r := &SubManager{
		Manager:           mgr,
		controllerBuilder: controllerBuilder,
		baseCfg:           rest.CopyConfig(cfg),
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
func (m *SubManager) Restart(gvks map[schema.GroupVersionKind]bool, apirs *discovery.APIInfo, force bool) error {
	if !force && !m.controllerBuilder.NeedsRestart(gvks) {
		return nil
	}
	ctx := m.context()

	var err error
	m.Manager, err = manager.New(rest.CopyConfig(m.baseCfg), manager.Options{})
	if err != nil {
		return errors.Wrap(err, "could not create the Manager for SubManager")
	}

	if err := m.controllerBuilder.UpdateScheme(m.GetScheme(), gvks); err != nil {
		return errors.Wrap(err, "could not update the scheme")
	}
	if err := m.controllerBuilder.StartControllers(ctx, m, gvks, apirs); err != nil {
		return errors.Wrap(err, "could not start controllers")
	}

	go func() {
		// Propagate errors with starting the SubManager up to the parent controller, so we can restart SubManager.
		m.errCh <- m.Start(ctx.Done())
	}()

	glog.Info("Starting SubManager")
	return nil
}
