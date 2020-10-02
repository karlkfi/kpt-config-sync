package parse

import (
	"time"

	"github.com/golang/glog"
	"github.com/google/nomos/pkg/importer/filesystem"
	"github.com/google/nomos/pkg/util/discovery"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// opts holds configuration and core functionality required by all
// parse.Runnables.
type opts struct {
	parser filesystem.ConfigParser

	// clusterName is the name of the cluster we're syncing configuration to.
	clusterName string

	// client knows how to read objects from a Kubernetes cluster and update status.
	client client.Client

	// pollingFrequency is how often to re-import configuration from the filesystem.
	//
	// For tests, use zero as it will poll continuously.
	pollingFrequency time.Duration

	// lastApplied is the directory (including git commit hash) last successfully
	// applied by the Applier.
	lastApplied string

	// discoveryInterface is how the parser learns what types are currently
	// available on the cluster.
	discoveryInterface discovery.ServerResourcer

	files
	updater
}

// TODO(b/167677315): This functionality should be on a DRY component of the
// root/namespace parsers rather than an "opts" struct.

// checkpoint marks the given string as the most recent checkpoint for state
// tracking and up-to-date checks.
func (o *opts) checkpoint(applied string) {
	glog.Infof("Reconciler checkpoint updated to %s", applied)
	o.lastApplied = applied
}

// invalidate clears the current checkpoint from state tracking.
func (o *opts) invalidate() {
	glog.Info("Reconciler checkpoint invalidated.")
	// Currently the only state that we track is the directory from the locally
	// mounted Git repo that was last applied fully.
	o.lastApplied = ""
}

// upToDate returns true if the given string matches the current checkpoint and
// if other components are up-to-date as well.
func (o *opts) upToDate(toApply string) bool {
	if o.lastApplied != toApply {
		glog.V(4).Infof("Reconciler checkpoint %s is not up-to-date with %s", o.lastApplied, toApply)
		return false
	}
	return !o.updater.needsUpdate()
}
