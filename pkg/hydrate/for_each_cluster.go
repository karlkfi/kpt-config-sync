package hydrate

import (
	"time"

	"github.com/google/nomos/pkg/importer/analyzer/transform/selectors"
	"github.com/google/nomos/pkg/importer/filesystem"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/util/namespaceconfig"
)

const (
	// We are guaranteed no Cluster's name is the empty string as that is a validation error.
	defaultCluster = ""
)

// ForEachCluster hydrates an AllConfigs for each declared cluster and executes the passed function
// on the result.
func ForEachCluster(
	p *filesystem.Parser,
	importToken string,
	loadTime time.Time,
	f func(clusterName string, configs *namespaceconfig.AllConfigs, err status.MultiError),
) {
	// Hydrate for empty string cluster name. This is the default configuration.
	defaultConfigs, err := p.Parse(importToken, &namespaceconfig.AllConfigs{}, loadTime, defaultCluster)
	f(defaultCluster, defaultConfigs, err)

	clusterRegistry := p.ReadClusterRegistryResources()
	clusters := selectors.FilterClusters(clusterRegistry)

	for _, cluster := range clusters {
		configs, err2 := p.Parse(importToken, &namespaceconfig.AllConfigs{}, loadTime, cluster.Name)
		f(cluster.Name, configs, err2)
	}
}
