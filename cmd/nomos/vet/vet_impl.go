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
	"github.com/google/nomos/pkg/importer/reader"
	"github.com/google/nomos/pkg/parse"
	"github.com/google/nomos/pkg/reconcilermanager"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/util/discovery"
	"github.com/google/nomos/pkg/vet"
	"github.com/pkg/errors"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
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
func runVet(root string, namespace string, sourceFormat filesystem.SourceFormat, skipAPIServer bool, allClusters bool, clusters []string) error {
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

	var syncedCRDs []*v1beta1.CustomResourceDefinition
	if !skipAPIServer {
		syncedCRDs, err = nomosparse.GetSyncedCRDs()
		if err != nil {
			return err
		}
	}

	var parser filesystem.ConfigParser
	switch sourceFormat {
	case filesystem.SourceFormatHierarchy:
		if namespace != "" {
			// The user could technically provide --source-format=unstructured.
			// This nuance isn't necessary to communicate nor confusing to omit.
			return errors.Errorf("if --%s is provided, --%s must be omitted",
				namespaceFlag, reconcilermanager.SourceFormat)
		}

		parser = filesystem.NewParser(&reader.File{}, !skipAPIServer)
		files = filesystem.FilterHierarchyFiles(rootDir, files)
	case filesystem.SourceFormatUnstructured:
		if namespace == "" {
			parser = filesystem.NewRawParser(&reader.File{}, !skipAPIServer, metav1.NamespaceDefault, declared.RootReconciler)
		} else {
			parser = parse.NewNamespace(&reader.File{}, !skipAPIServer, declared.Scope(namespace))
		}
	default:
		return fmt.Errorf("unknown %s value %q", reconcilermanager.SourceFormat, sourceFormat)
	}

	filePaths := reader.FilePaths{
		RootDir:   rootDir,
		PolicyDir: cmpath.RelativeOS(root),
		Files:     files,
	}

	addFunc := vet.AddCachedAPIResources(rootDir.Join(vet.APIResourcesPath))
	builder := discovery.ScoperBuilder(dc, addFunc)

	var runKptfileExistenceValidator bool
	if namespace == "" {
		// Kptfile is not supported in root repos, set runKptfileExistenceValidator to true on root repos.
		// TODO(b/172610552): After the support for Kptfile in a root repo is added, this validator will no longer be needed.
		runKptfileExistenceValidator = true
	}

	// Track per-cluster vet errors.
	var vetErrs []string
	hydrate.ForEachCluster(parser, syncedCRDs, builder, filePaths,
		runKptfileExistenceValidator, vetCluster(&vetErrs, allClusters, clusters),
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
