package vet

import (
	"os"

	"github.com/google/nomos/cmd/nomos/flags"
	"github.com/google/nomos/cmd/nomos/parse"
	"github.com/google/nomos/cmd/nomos/util"
	"github.com/google/nomos/pkg/importer/filesystem"
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
		clusterName := os.Getenv("CLUSTER_NAME")

		_, err := parse.Parse(clusterName,
			filesystem.ParserOpt{Validate: flags.Validate, Vet: true, Extension: &filesystem.NomosVisitorProvider{}, RootPath: rootPath})
		if err != nil {
			util.PrintErrAndDie(err)
		}
	},
}
