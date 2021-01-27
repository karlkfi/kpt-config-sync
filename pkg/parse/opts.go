package parse

import (
	"context"
	"time"

	"github.com/golang/glog"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/filesystem"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/util/discovery"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// opts holds configuration and core functionality required by all parsers.
type opts struct {
	parser filesystem.ConfigParser

	// clusterName is the name of the cluster we're syncing configuration to.
	clusterName string

	// client knows how to read objects from a Kubernetes cluster and update status.
	client client.Client

	// reconcilerName is the name of the reconciler resources, such as service account, service, deployment and etc.
	reconcilerName string

	// pollingFrequency is how often to re-import configuration from the filesystem.
	//
	// For tests, use zero as it will poll continuously.
	pollingFrequency time.Duration

	// ResyncPeriod is the period of time between forced re-sync from Git (even
	// without a new commit).
	resyncPeriod time.Duration

	// discoveryInterface is how the parser learns what types are currently
	// available on the cluster.
	discoveryInterface discovery.ServerResourcer

	// lastApplied keeps the state for the last successful-applied policyDir.
	lastApplied string

	files
	updater
}

// Parser represents a parser that can be pointed at and continuously parse
// a git repository.
type Parser interface {
	parseSource(ctx context.Context, state gitState) ([]core.Object, status.MultiError)
	setSourceStatus(ctx context.Context, state gitState, errs status.MultiError) error
	setSyncStatus(ctx context.Context, commit string, errs status.MultiError) error
	options() *opts
}

// checkpoint marks the given string as the most recent checkpoint for state
// tracking and up-to-date checks if `applied` has not been checkpointed.
func (o *opts) checkpoint(applied string) {
	if applied != o.lastApplied {
		glog.Infof("Reconciler checkpoint updated to %s", applied)
		o.lastApplied = applied
		o.cache.needToRetry = false
	}
}

// invalidate logs the errors, clears the state tracking information and sets needToRetry to true.
// invalidate does not clean up the cache.
func (o *opts) invalidate(err status.MultiError) {
	glog.Error(err)
	glog.Info("Reconciler checkpoint invalidated.")
	o.lastApplied = ""
	o.cache.needToRetry = true
}

func (o *opts) resetCache() {
	o.cache = cache{}
}
