package vet

import (
	"fmt"
	"path/filepath"
	"strings"

	nomosparse "github.com/google/nomos/cmd/nomos/parse"
	"github.com/google/nomos/pkg/declared"
	"github.com/google/nomos/pkg/hydrate"
	"github.com/google/nomos/pkg/importer"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/filesystem"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/parse"
	"github.com/google/nomos/pkg/status"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// vet runs nomos vet with the specified options.
//
// root is the OS-specific path to the Nomos policy root.
//   If relative, it is assumed to be relative to the working directory.
// namespace, if non-emptystring, validates the repo as a CSMR Namespace
//   repository.
// sourceFormat is whether the repository is in the hierarchy or unstructured
//   format.
// skipAPIServer is whether to skip the API Server checks.
// allClusters is whether we are implicitly vetting every cluster.
// clusters is the set of clusters we are checking.
//   Only used if allClusters is false.
func vet(root string, namespace string, sourceFormat filesystem.SourceFormat, skipAPIServer bool, allClusters bool, clusters []string) error {
	root, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	rootDir, err := cmpath.AbsoluteOS(root)
	if err != nil {
		return err
	}
	rootDir, err = rootDir.EvalSymlinks()
	if err != nil {
		return err
	}

	files, err := nomosparse.FindFiles(rootDir)
	if err != nil {
		return err
	}

	dc, err := importer.DefaultCLIOptions.ToDiscoveryClient()
	if err != nil {
		return err
	}

	if sourceFormat == "" {
		if namespace == "" {
			// Default to hierarchical if --namespace is not provided.
			sourceFormat = filesystem.SourceFormatHierarchy
		} else {
			// Default to unstructured if --namespace is provided.
			sourceFormat = filesystem.SourceFormatUnstructured
		}
	}

	getSyncedCRDs := nomosparse.GetSyncedCRDs
	if skipAPIServer {
		getSyncedCRDs = filesystem.NoSyncedCRDs
	}

	var parser filesystem.ConfigParser
	switch sourceFormat {
	case filesystem.SourceFormatHierarchy:
		if namespace != "" {
			// The user could technically provide --source-format=unstructured.
			// This nuance isn't necessary to communicate nor confusing to omit.
			return errors.Errorf("if --%s is provided, --%s must be omitted",
				namespaceFlag, sourceFormatFlag)
		}

		parser = filesystem.NewParser(&filesystem.FileReader{}, dc)
		files = filesystem.FilterHierarchyFiles(rootDir, files)
	case filesystem.SourceFormatUnstructured:
		if namespace == "" {
			parser = filesystem.NewRawParser(&filesystem.FileReader{}, dc, metav1.NamespaceDefault)
		} else {
			parser = parse.NewNamespace(&filesystem.FileReader{}, dc, declared.Scope(namespace))
		}
	default:
		return fmt.Errorf("unknown %s value %q", sourceFormatFlag, sourceFormat)
	}

	filePaths := filesystem.FilePaths{
		RootDir:   rootDir,
		PolicyDir: cmpath.RelativeOS(root),
		Files:     files,
	}

	// Track per-cluster vet errors.
	var vetErrs []string
	hydrate.ForEachCluster(parser, getSyncedCRDs, !skipAPIServer, filePaths,
		vetCluster(&vetErrs, allClusters, clusters),
	)
	if len(vetErrs) > 0 {
		return errors.New(strings.Join(vetErrs, "\n\n"))
	}
	return nil
}

// clusterErrors is the set of vet errors for a specific Cluster.
type clusterErrors struct {
	name string
	status.MultiError
}

func (e clusterErrors) Error() string {
	if e.name == "defaultcluster" {
		return e.MultiError.Error()
	}
	return fmt.Sprintf("errors for cluster %q:\n%v\n", e.name, e.MultiError.Error())
}

func vetCluster(vetErrors *[]string, allClusters bool, clusters []string) func(clusterName string, fileObjects []ast.FileObject, errs status.MultiError) {
	return func(clusterName string, _ []ast.FileObject, errs status.MultiError) {
		clusterEnabled := allClusters
		for _, cluster := range clusters {
			if clusterName == cluster {
				clusterEnabled = true
			}
		}
		if !clusterEnabled {
			return
		}

		if errs != nil {
			if clusterName == "" {
				clusterName = nomosparse.UnregisteredCluster
			}
			*vetErrors = append(*vetErrors, clusterErrors{
				name:       clusterName,
				MultiError: errs,
			}.Error())
		}
	}
}
