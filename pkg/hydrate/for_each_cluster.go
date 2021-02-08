package hydrate

import (
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/transform/selectors"
	"github.com/google/nomos/pkg/importer/filesystem"
	"github.com/google/nomos/pkg/importer/reader"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/util/discovery"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
)

const (
	// We assume users will not name any cluster "defaultcluster".
	defaultCluster = "defaultcluster"
)

// ForEachCluster hydrates an AllConfigs for each declared cluster and executes the passed function
// on the result.
//
// parser is the ConfigParser which returns a set of FileObjects and a possible MultiError
//   when Parse is called.
// getSyncedCRDs is the set of CRDs synced the the cluster used for APIServer checks.
// enableAPIServerChecks is whether to call Parse with APIServer checks enabled.
// apiResources is how to read cached API resources from the disk.
// filePaths is the list of absolute file paths to parse and the absolute and
//   relative paths of the Nomos root.
// f is a function with three arguments:
//    clusterName, the name of the Cluster the Parser was called with.
//    fileObjects, the FileObjects which Parser.Parse returned.
//    err, the MultiError which Parser.Parse returned, if there was one.
//
// Per standard ForEach conventions, ForEachCluster has no return value.
func ForEachCluster(parser filesystem.ConfigParser, syncedCRDs []*v1beta1.CustomResourceDefinition, buildScoper discovery.BuildScoperFunc, filePaths reader.FilePaths, f func(clusterName string, fileObjects []ast.FileObject, err status.MultiError)) {
	clusterRegistry := parser.ReadClusterRegistryResources(filePaths)
	clusters, errs := selectors.FilterClusters(clusterRegistry)

	// Hydrate for empty string cluster name. This is the default configuration.
	defaultFileObjects, err2 := parser.Parse(defaultCluster, syncedCRDs, buildScoper, filePaths)
	f(defaultCluster, defaultFileObjects, status.Append(errs, err2))

	for _, cluster := range clusters {
		fileObjects, errs := parser.Parse(cluster.Name, syncedCRDs, buildScoper, filePaths)
		f(cluster.Name, fileObjects, errs)
	}
}
