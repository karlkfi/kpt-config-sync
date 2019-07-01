package vet

import (
	"os"
	"time"

	"github.com/google/nomos/cmd/nomos/flags"
	"github.com/google/nomos/cmd/nomos/parse"
	"github.com/google/nomos/cmd/nomos/util"
	"github.com/google/nomos/pkg/hydrate"
	"github.com/google/nomos/pkg/importer/filesystem"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/util/namespaceconfig"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func init() {
	flags.AddPath(Cmd)
	flags.AddValidate(Cmd)
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

		opts := filesystem.ParserOpt{Validate: flags.Validate, Vet: true, Extension: &filesystem.NomosVisitorProvider{}, RootPath: rootPath}
		parser, err := parse.NewParser(opts)
		if err != nil {
			util.PrintErrAndDie(err)
		}

		// TODO: Allow choosing which clusters to show errors for.
		encounteredError := false
		hydrate.ForEachCluster(parser, "", time.Time{}, func(clusterName string, configs *namespaceconfig.AllConfigs, err status.MultiError) {
			if err != nil {
				util.PrintErrOrDie(errors.Wrapf(err, "errors in Cluster %q", clusterName))
				encounteredError = true
			}
		})
		if encounteredError {
			os.Exit(1)
		}
	},
}
