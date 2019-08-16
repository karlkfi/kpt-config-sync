package vet

import (
	"fmt"
	"os"
	"time"

	"github.com/google/nomos/cmd/nomos/flags"
	"github.com/google/nomos/cmd/nomos/parse"
	"github.com/google/nomos/cmd/nomos/util"
	"github.com/google/nomos/pkg/api/configmanagement"
	"github.com/google/nomos/pkg/hydrate"
	"github.com/google/nomos/pkg/importer"
	"github.com/google/nomos/pkg/importer/filesystem"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/util/namespaceconfig"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	useHierarchyFlag = "use-hierarchy"

	useHierarchy bool
)

func init() {
	flags.AddClusters(Cmd)
	flags.AddPath(Cmd)
	Cmd.Flags().BoolVar(&useHierarchy, useHierarchyFlag, true,
		fmt.Sprintf("If true, validate as a %s Repo.\n"+
			"If false, validate recursively as a directory of manifests.", configmanagement.ProductName))
}

// Cmd is the Cobra object representing the nomos vet command.
var Cmd = &cobra.Command{
	Use:   "vet",
	Short: "Validate a Anthos Configuration Management directory",
	Long: `Validate a Anthos Configuration Management directory
Checks for semantic and syntactic errors in a Anthos Configuration Management directory
that will interfere with applying resources. Prints found errors to STDERR and
returns a non-zero error code if any issues are found.
`,
	Example: `  nomos vet
  nomos vet --path=my/directory
  nomos vet --path=/path/to/my/directory`,
	Args: cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		rootDir := flags.Path.String()
		rootPath := util.GetRootOrDie(rootDir)

		var parser filesystem.ConfigParser
		if useHierarchy {
			opts := filesystem.ParserOpt{Extension: &filesystem.NomosVisitorProvider{}, RootPath: rootPath}
			var err error
			parser, err = parse.NewParser(opts)
			if err != nil {
				util.PrintErrAndDie(err)
			}
		} else {
			parser = filesystem.NewRawParser(rootPath.Join(cmpath.FromSlash(".")), &filesystem.FileReader{ClientGetter: importer.DefaultCLIOptions}, importer.DefaultCLIOptions)
		}

		encounteredError := false
		hydrate.ForEachCluster(parser, "", time.Time{}, func(clusterName string, _ *namespaceconfig.AllConfigs, err status.MultiError) {
			clusterEnabled := flags.AllClusters()
			for _, cluster := range flags.Clusters {
				if clusterName == cluster {
					clusterEnabled = true
				}
			}
			if !clusterEnabled {
				return
			}

			if err != nil {
				if clusterName == "" {
					clusterName = parse.UnregisteredCluster
				}
				util.PrintErrOrDie(errors.Wrapf(err, "errors for Cluster %q", clusterName))
				encounteredError = true
			}
		})
		if encounteredError {
			os.Exit(1)
		}
	},
}
