package hydrate

import (
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/transform/selectors"
	"github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical"
	"github.com/google/nomos/pkg/importer/filesystem"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/importer/reader"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/vet"
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
// runKptfileExistenceValidator is whether to run KptfileExistenceValidator.
// f is a function with three arguments:
//    clusterName, the name of the Cluster the Parser was called with.
//    fileObjects, the FileObjects which Parser.Parse returned.
//    err, the MultiError which Parser.Parse returned, if there was one.
//
// Per standard ForEach conventions, ForEachCluster has no return value.
func ForEachCluster(parser filesystem.ConfigParser, getSyncedCRDs filesystem.GetSyncedCRDs, enableAPIServerChecks bool, apiResources cmpath.Absolute, filePaths reader.FilePaths, runKptfileExistenceValidator bool, f func(clusterName string, fileObjects []ast.FileObject, err status.MultiError)) {
	// Hydrate for empty string cluster name. This is the default configuration.
	defaultCoreObjects, err := parser.Parse(defaultCluster, enableAPIServerChecks, vet.AddCachedAPIResources(apiResources), getSyncedCRDs, filePaths)
	defaultFileObjects := filesystem.AsFileObjects(defaultCoreObjects)
	f(defaultCluster, defaultFileObjects, err)

	clusterRegistry := parser.ReadClusterRegistryResources(filePaths)
	clusters := selectors.FilterClusters(clusterRegistry)

	for _, cluster := range clusters {
		coreObjects, err2 := parser.Parse(cluster.Name, enableAPIServerChecks, vet.AddCachedAPIResources(apiResources), getSyncedCRDs, filePaths)
		fileObjects := filesystem.AsFileObjects(coreObjects)

		// TODO(b/172610552): After the support for Kptfile in a root repo is added, this validator will no longer be needed.
		if runKptfileExistenceValidator {
			err2 = status.Append(err2, nonhierarchical.KptfileExistenceValidator.Validate(fileObjects))
		}
		f(cluster.Name, fileObjects, err2)
	}
}
