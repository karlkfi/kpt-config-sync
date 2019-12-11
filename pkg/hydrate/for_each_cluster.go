package hydrate

import (
	"github.com/google/nomos/pkg/importer/analyzer/transform/selectors"
	"github.com/google/nomos/pkg/importer/filesystem"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/util/namespaceconfig"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// We assume users will not name any cluster "defaultcluster".
	defaultCluster = "defaultcluster"
)

// ForEachCluster hydrates an AllConfigs for each declared cluster and executes the passed function
// on the result.
func ForEachCluster(
	p filesystem.ConfigParser,
	importToken string,
	loadTime metav1.Time,
	f func(clusterName string, configs *namespaceconfig.AllConfigs, err status.MultiError),
) {
	// Hydrate for empty string cluster name. This is the default configuration.
	defaultFileObjects, err := p.Parse(&namespaceconfig.AllConfigs{}, defaultCluster)
	defaultConfigs := namespaceconfig.NewAllConfigs(importToken, loadTime, defaultFileObjects)
	f(defaultCluster, defaultConfigs, err)

	clusterRegistry := p.ReadClusterRegistryResources()
	clusters := selectors.FilterClusters(clusterRegistry)

	for _, cluster := range clusters {
		fileObjects, err2 := p.Parse(&namespaceconfig.AllConfigs{}, cluster.Name)
		configs := namespaceconfig.NewAllConfigs(importToken, loadTime, fileObjects)
		f(cluster.Name, configs, err2)
	}
}
