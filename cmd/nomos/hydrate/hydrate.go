package hydrate

import (
	"os"
	"path/filepath"

	"github.com/google/nomos/cmd/nomos/flags"
	"github.com/google/nomos/cmd/nomos/parse"
	"github.com/google/nomos/cmd/nomos/util"
	"github.com/google/nomos/pkg/declared"
	"github.com/google/nomos/pkg/hydrate"
	"github.com/google/nomos/pkg/importer"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/filesystem"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/importer/reader"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/util/discovery"
	"github.com/google/nomos/pkg/validate"
	"github.com/google/nomos/pkg/vet"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	flat      bool
	outPath   string
	extension string
)

func init() {
	flags.AddClusters(Cmd)
	flags.AddPath(Cmd)
	flags.AddSkipAPIServerCheck(Cmd)

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
	Cmd.Flags().StringVar(&extension, "format", "yaml",
		`Output format of hydrated configuration. Accepts 'yaml' and 'json'.`)
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

		switch extension {
		case "yaml", "json": // do nothing
		default:
			return errors.New("format must argument be 'yaml' or 'json'")
		}

		abs, err := filepath.Abs(flags.Path)
		if err != nil {
			return err
		}
		rootDir, err := cmpath.AbsoluteOS(abs)
		if err != nil {
			return err
		}
		policyDir := cmpath.RelativeOS(flags.Path)

		files, err := parse.FindFiles(rootDir)
		if err != nil {
			return err
		}
		// Hydrate is only used in hierarchical mode.
		files = filesystem.FilterHierarchyFiles(rootDir, files)

		filePaths := reader.FilePaths{
			RootDir:   rootDir,
			PolicyDir: policyDir,
			Files:     files,
		}

		parser := filesystem.NewParser(&reader.File{})

		crds, err := parse.GetSyncedCRDs(cmd.Context(), flags.SkipAPIServer)
		if err != nil {
			return err
		}
		addFunc := vet.AddCachedAPIResources(rootDir.Join(vet.APIResourcesPath))

		var serverResourcer discovery.ServerResourcer = discovery.NoOpServerResourcer{}
		var converter *declared.ValueConverter
		if !flags.SkipAPIServer {
			dc, err := importer.DefaultCLIOptions.ToDiscoveryClient()
			if err != nil {
				return err
			}
			serverResourcer = dc

			converter, err = declared.NewValueConverter(dc)
			if err != nil {
				return err
			}
		}

		options := validate.Options{
			PolicyDir:         policyDir,
			PreviousCRDs:      crds,
			BuildScoper:       discovery.ScoperBuilder(serverResourcer, addFunc),
			Converter:         converter,
			AllowUnknownKinds: flags.SkipAPIServer,
		}

		var allObjects []ast.FileObject
		encounteredError := false
		numClusters := 0
		hydrate.ForEachCluster(parser, options, filePaths, func(clusterName string, fileObjects []ast.FileObject, err status.MultiError) {
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
					clusterName = parse.UnregisteredCluster
				}
				util.PrintErrOrDie(errors.Wrapf(err, "errors for Cluster %q", clusterName))
				encounteredError = true
				return
			}

			allObjects = append(allObjects, fileObjects...)
		})

		multiCluster := numClusters > 1
		fileObjects := hydrate.GenerateUniqueFileNames(extension, multiCluster, allObjects...)
		hydrate.Clean(fileObjects)
		if flat {
			err = hydrate.PrintFlatOutput(outPath, extension, fileObjects)
		} else {
			err = hydrate.PrintDirectoryOutput(outPath, extension, fileObjects)
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
