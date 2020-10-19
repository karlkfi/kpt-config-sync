package vet

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/google/nomos/cmd/nomos/flags"
	"github.com/google/nomos/cmd/nomos/parse"
	"github.com/google/nomos/pkg/hydrate"
	"github.com/google/nomos/pkg/importer"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/filesystem"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// vet runs nomos vet with the specified options.
//
// root is the OS-specific path to the Nomos policy root.
//   If relative, it is assumed to be relative to the working directory.
// sourceFormat is whether the repository is in the hierarchy or unstructured
//   format.
// skipAPIServer is whether to skip the API Server checks.
// allClusters is whether we are implicitly vetting every cluster.
// clusters is the set of clusters we are checking.
//   Only used if allClusters is false.
func vet(
	root string,
	sourceFormat string,
	skipAPIServer bool,
	allClusters bool,
	clusters []string,
) error {
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

	files, err := parse.FindFiles(rootDir)
	if err != nil {
		return err
	}

	dc, err := importer.DefaultCLIOptions.ToDiscoveryClient()
	if err != nil {
		return err
	}

	var parser filesystem.ConfigParser
	switch sourceFormat {
	case hierarchyFormat:
		parser = parse.NewParser(dc)
		files = filesystem.FilterHierarchyFiles(rootDir, files)
	case unstructuredFormat:
		parser = filesystem.NewRawParser(&filesystem.FileReader{}, dc, metav1.NamespaceDefault)
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
	hydrate.ForEachCluster(parser, parse.GetSyncedCRDs, !skipAPIServer, filePaths,
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
	errs := e.MultiError.Errors()
	if len(errs) == 1 && errs[0].Code() == status.APIServerErrorCode {
		return fmt.Sprintf("did you mean to run with --%s?: %v", flags.SkipAPIServerFlag, e.Error())
	}
	return fmt.Sprintf("errors for cluster %q: %v\n", e.name, e.MultiError.Error())
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
				clusterName = parse.UnregisteredCluster
			}
			*vetErrors = append(*vetErrors, clusterErrors{
				name:       clusterName,
				MultiError: errs,
			}.Error())
		}
	}
}
