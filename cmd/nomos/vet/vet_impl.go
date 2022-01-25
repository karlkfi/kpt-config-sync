package vet

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/google/nomos/cmd/nomos/flags"
	nomosparse "github.com/google/nomos/cmd/nomos/parse"
	"github.com/google/nomos/cmd/nomos/util"
	"github.com/google/nomos/pkg/declared"
	"github.com/google/nomos/pkg/hydrate"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/filesystem"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/importer/reader"
	"github.com/google/nomos/pkg/parse"
	"github.com/google/nomos/pkg/reconcilermanager"
	"github.com/google/nomos/pkg/status"
	"github.com/pkg/errors"
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
func runVet(ctx context.Context, namespace string, sourceFormat filesystem.SourceFormat) error {
	if sourceFormat == "" {
		if namespace == "" {
			// Default to hierarchical if --namespace is not provided.
			sourceFormat = filesystem.SourceFormatHierarchy
		} else {
			// Default to unstructured if --namespace is provided.
			sourceFormat = filesystem.SourceFormatUnstructured
		}
	}

	rootDir, needsHydrate, err := hydrate.ValidateHydrateFlags(sourceFormat)
	if err != nil {
		return err
	}

	if needsHydrate {
		// update rootDir to point to the hydrated output for further processing.
		if rootDir, err = hydrate.ValidateAndRunKustomize(rootDir.OSPath()); err != nil {
			return err
		}
		// delete the hydrated output directory in the end.
		defer func() {
			_ = os.RemoveAll(rootDir.OSPath())
		}()
	}

	files, err := nomosparse.FindFiles(rootDir)
	if err != nil {
		return err
	}

	parser := filesystem.NewParser(&reader.File{})

	options, err := hydrate.ValidateOptions(ctx, rootDir)
	if err != nil {
		return err
	}

	switch sourceFormat {
	case filesystem.SourceFormatHierarchy:
		if namespace != "" {
			// The user could technically provide --source-format=unstructured.
			// This nuance isn't necessary to communicate nor confusing to omit.
			return errors.Errorf("if --namespace is provided, --%s must be omitted or set to %s",
				reconcilermanager.SourceFormat, filesystem.SourceFormatUnstructured)
		}

		files = filesystem.FilterHierarchyFiles(rootDir, files)
	case filesystem.SourceFormatUnstructured:
		if namespace == "" {
			options = parse.OptionsForScope(options, declared.RootReconciler)
		} else {
			options = parse.OptionsForScope(options, declared.Scope(namespace))
		}
	default:
		return fmt.Errorf("unknown %s value %q", reconcilermanager.SourceFormat, sourceFormat)
	}

	filePaths := reader.FilePaths{
		RootDir:   rootDir,
		PolicyDir: cmpath.RelativeOS(rootDir.OSPath()),
		Files:     files,
	}

	// Track per-cluster vet errors.
	var allObjects []ast.FileObject
	var vetErrs []string
	numClusters := 0
	hydrate.ForEachCluster(parser, options, sourceFormat, filePaths, func(clusterName string, fileObjects []ast.FileObject, err status.MultiError) {
		clusterEnabled := flags.AllClusters()
		for _, cluster := range flags.Clusters {
			if clusterName == cluster {
				clusterEnabled = true
			}
		}
		if !clusterEnabled {
			return
		}
		numClusters++

		if err != nil {
			if clusterName == "" {
				clusterName = nomosparse.UnregisteredCluster
			}
			vetErrs = append(vetErrs, clusterErrors{
				name:       clusterName,
				MultiError: err,
			}.Error())
		}

		if keepOutput {
			allObjects = append(allObjects, fileObjects...)
		}
	})
	if keepOutput {
		multiCluster := numClusters > 1
		fileObjects := hydrate.GenerateFileObjects(multiCluster, allObjects...)
		if err := hydrate.PrintDirectoryOutput(outPath, flags.OutputFormat, fileObjects); err != nil {
			_ = util.PrintErr(err)
		}
	}
	if len(vetErrs) > 0 {
		return errors.New(strings.Join(vetErrs, "\n\n"))
	}

	fmt.Println("✅ No validation issues found.")
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
