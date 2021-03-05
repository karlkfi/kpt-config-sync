package hydrate

import (
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/transform/selectors"
	"github.com/google/nomos/pkg/importer/filesystem"
	"github.com/google/nomos/pkg/importer/reader"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/validate"
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
func ForEachCluster(parser filesystem.ConfigParser, options validate.Options, filePaths reader.FilePaths, f func(clusterName string, fileObjects []ast.FileObject, err status.MultiError)) {
	isHierarchical := options.DefaultNamespace == "" && !options.IsNamespaceReconciler
	clusterRegistry, errs := parser.ReadClusterRegistryResources(filePaths)
	clusters, err2 := selectors.FilterClusters(clusterRegistry)
	errs = status.Append(errs, err2)

	// Hydrate for empty string cluster name. This is the default configuration.
	options.ClusterName = defaultCluster
	defaultFileObjects, err2 := parser.Parse(filePaths)
	errs = status.Append(errs, err2)

	if isHierarchical {
		defaultFileObjects, err2 = validate.Hierarchical(defaultFileObjects, options)
	} else {
		defaultFileObjects, err2 = validate.Unstructured(defaultFileObjects, options)
	}
	errs = status.Append(errs, err2)

	f(defaultCluster, defaultFileObjects, errs)

	for _, cluster := range clusters {
		options.ClusterName = cluster.Name
		fileObjects, errs := parser.Parse(filePaths)

		if isHierarchical {
			fileObjects, err2 = validate.Hierarchical(fileObjects, options)
		} else {
			fileObjects, err2 = validate.Unstructured(fileObjects, options)
		}

		errs = status.Append(errs, err2)
		f(cluster.Name, fileObjects, errs)
	}
}
