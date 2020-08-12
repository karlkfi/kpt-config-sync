package hydrate

import (
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/transform/selectors"
	"github.com/google/nomos/pkg/importer/filesystem"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/status"
)

const (
	// We assume users will not name any cluster "defaultcluster".
	defaultCluster = "defaultcluster"
)

// ForEachCluster hydrates an AllConfigs for each declared cluster and executes the passed function
// on the result.
//
// p is the ConfigParser which returns a set of FileObjects and a possible MultiError
//   when Parse is called.
// syncedCRDs is the set of CRDs synced the the cluster used for APIServer checks.
// enableAPIServerChecks is whether to call Parse with APIServer checks enabled.
// f is a function with three arguments:
//    clusterName, the name of the Cluster the Parser was called with.
//    fileObjects, the FileObjects which Parser.Parse returned.
//    err, the MultiError which Parser.Parse returned, if there was one.
//
// Per standard ForEach conventions, ForEachCluster has no return value.
func ForEachCluster(
	parser filesystem.ConfigParser,
	getSyncedCRDs filesystem.GetSyncedCRDs,
	enableAPIServerChecks bool,
	rootDir cmpath.Absolute,
	files []cmpath.Absolute,
	f func(clusterName string, fileObjects []ast.FileObject, err status.MultiError),
) {
	// Hydrate for empty string cluster name. This is the default configuration.
	defaultCoreObjects, err := parser.Parse(defaultCluster, enableAPIServerChecks, getSyncedCRDs, rootDir, files)
	defaultFileObjects := filesystem.AsFileObjects(defaultCoreObjects)
	f(defaultCluster, defaultFileObjects, err)

	clusterRegistry := parser.ReadClusterRegistryResources(rootDir, files)
	clusters := selectors.FilterClusters(clusterRegistry)

	for _, cluster := range clusters {
		coreObjects, err2 := parser.Parse(cluster.Name, enableAPIServerChecks, getSyncedCRDs, rootDir, files)
		fileObjects := filesystem.AsFileObjects(coreObjects)
		f(cluster.Name, fileObjects, err2)
	}
}
