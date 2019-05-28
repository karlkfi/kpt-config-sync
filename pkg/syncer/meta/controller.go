// Package meta includes controllers responsible for managing other controllers based on Syncs and CRDs.
package meta

import (
	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/syncer/crd"
	"github.com/google/nomos/pkg/syncer/sync"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// AddControllers adds all controllers that manage other controllers.
func AddControllers(mgr manager.Manager, enableCRDs bool) error {
	// Set up Scheme for nomos resources.
	if err := v1.AddToScheme(mgr.GetScheme()); err != nil {
		return err
	}

	rc := sync.NewRestartChannel(make(chan event.GenericEvent))
	if err := sync.AddController(mgr, rc); err != nil {
		return err
	}
	if enableCRDs {
		return crd.AddCRDController(mgr, sync.RestartSignal(rc))
	}
	return nil
}
