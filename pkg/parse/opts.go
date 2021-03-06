package parse

import (
	"context"
	"time"

	"github.com/google/nomos/pkg/declared"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
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

	// converter uses the discoveryInterface to encode the declared fields of
	// objects in Git.
	converter *declared.ValueConverter

	files
	updater
}

// Parser represents a parser that can be pointed at and continuously parse
// a git repository.
type Parser interface {
	parseSource(ctx context.Context, state gitState) ([]ast.FileObject, status.MultiError)
	setSourceStatus(ctx context.Context, oldStatus, newStatus gitStatus) error
	setSyncStatus(ctx context.Context, oldStatus, newStatus gitStatus) error
	options() *opts
}

func (o *opts) k8sClient() client.Client {
	return o.client
}

func (o *opts) discoveryClient() discovery.ServerResourcer {
	return o.discoveryInterface
}
