package vet

import (
	"fmt"
	"os"

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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	disableHierarchyFlag = "disable-hierarchy"

	disableHierarchy bool
)

func init() {
	flags.AddClusters(Cmd)
	flags.AddPath(Cmd)
	Cmd.Flags().BoolVar(&disableHierarchy, disableHierarchyFlag, false,
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
		if disableHierarchy {
			parser = filesystem.NewRawParser(rootPath.Join(cmpath.FromSlash(".")), &filesystem.FileReader{}, importer.DefaultCLIOptions)
		} else {
			opts := filesystem.ParserOpt{Extension: &filesystem.NomosVisitorProvider{}, RootPath: rootPath}
			var err error
			parser, err = parse.NewParser(opts)
			if err != nil {
				util.PrintErrAndDie(err)
			}
		}

		encounteredError := false
		hydrate.ForEachCluster(parser, "", metav1.Time{}, func(clusterName string, _ *namespaceconfig.AllConfigs, err status.MultiError) {
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
