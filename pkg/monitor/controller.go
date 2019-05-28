// Package monitor contains the controller for monitoring the state of Nomos on a cluster.
package monitor

import (
	"github.com/google/nomos/clientgen/apis/scheme"
	"github.com/google/nomos/pkg/monitor/clusterconfig"
	"github.com/google/nomos/pkg/monitor/namespaceconfig"
	"github.com/google/nomos/pkg/monitor/state"
	"github.com/google/nomos/pkg/syncer/client"
	"github.com/google/nomos/pkg/util/repo"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// AddToManager adds all Controllers to the Manager
func AddToManager(mgr manager.Manager) error {
	if err := scheme.AddToScheme(mgr.GetScheme()); err != nil {
		return errors.Wrapf(err, "pkg/monitor.AddToManager")
	}
	syncCl := client.New(mgr.GetClient(), nil)
	repoCl := repo.New(syncCl)

	cs := state.NewClusterState()
	if err := clusterconfig.AddController(mgr, repoCl, cs); err != nil {
		return err
	}
	if err := namespaceconfig.AddController(mgr, repoCl, cs); err != nil {
		return err
	}
	return nil
}
