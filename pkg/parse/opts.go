package parse

import (
	"time"

	"github.com/google/nomos/pkg/importer/filesystem"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// opts holds configuration and core functionality required by all
// parse.Runnables.
type opts struct {
	parser filesystem.ConfigParser

	// clusterName is the name of the cluster we're syncing configuration to.
	clusterName string

	// reader knows how to read objects from a Kubernetes cluster.
	reader client.Reader

	// pollingFrequency is how often to re-import configuration from the filesystem.
	//
	// For tests, use zero as it will poll continuously.
	pollingFrequency time.Duration

	files
	updater
}
