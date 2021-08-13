package hydrate

import (
	"os"

	"github.com/google/nomos/cmd/nomos/flags"
	nomosparse "github.com/google/nomos/cmd/nomos/parse"
	"github.com/google/nomos/cmd/nomos/util"
	"github.com/google/nomos/pkg/hydrate"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/filesystem"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/importer/reader"
	"github.com/google/nomos/pkg/status"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	flat    bool
	outPath string
)

func init() {
	flags.AddClusters(Cmd)
	flags.AddPath(Cmd)
	flags.AddSkipAPIServerCheck(Cmd)
	flags.AddSourceFormat(Cmd)
	flags.AddOutputFormat(Cmd)
	Cmd.Flags().BoolVar(&flat, "flat", false,
		`If enabled, print all output to a single file`)
	Cmd.Flags().StringVar(&outPath, "output", "compiled",
		`Location to write hydrated configuration to.

If --flat is not enabled, writes each resource manifest as a
separate file. You may run "kubectl apply -fR" on the result to apply
the configuration to a cluster. If the repository declares any Cluster
resources, contains a subdirectory for each Cluster.

If --flat is enabled, writes to the, writes a single file holding all
resource manifests. You may run "kubectl apply -f" on the result to
apply the configuration to a cluster.`)
}

// Cmd is the Cobra object representing the hydrate command.
var Cmd = &cobra.Command{
	Use:   "hydrate",
	Short: "Compiles the local repository to the exact form that would be sent to the APIServer.",
	Long: `Compiles the local repository to the exact form that would be sent to the APIServer.

The output directory consists of one directory per declared Cluster, and defaultcluster/ for
clusters without declarations. Each directory holds the full set of configs for a single cluster,
which you could kubectl apply -fR to the cluster, or have Config Sync sync to the cluster.`,
	Args: cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Don't show usage on error, as argument validation passed.
		cmd.SilenceUsage = true

		sourceFormat := filesystem.SourceFormat(flags.SourceFormat)
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

		options, err := hydrate.ValidateOptions(cmd.Context(), rootDir)
		if err != nil {
			return err
		}

		if sourceFormat == filesystem.SourceFormatHierarchy {
			files = filesystem.FilterHierarchyFiles(rootDir, files)
		}

		filePaths := reader.FilePaths{
			RootDir:   rootDir,
			PolicyDir: cmpath.RelativeOS(rootDir.OSPath()),
			Files:     files,
		}

		var allObjects []ast.FileObject
		encounteredError := false
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
				util.PrintErrOrDie(errors.Wrapf(err, "errors for Cluster %q", clusterName))

				encounteredError = true

				if status.HasActionableErrors(err) {
					return
				}
			}

			allObjects = append(allObjects, fileObjects...)
		})

		multiCluster := numClusters > 1
		fileObjects := hydrate.GenerateFileObjects(multiCluster, allObjects...)
		if flat {
			err = hydrate.PrintFlatOutput(outPath, flags.OutputFormat, fileObjects)
		} else {
			err = hydrate.PrintDirectoryOutput(outPath, flags.OutputFormat, fileObjects)
		}
		if err != nil {
			return err
		}

		if encounteredError {
			os.Exit(1)
		}

		return nil
	},
}
