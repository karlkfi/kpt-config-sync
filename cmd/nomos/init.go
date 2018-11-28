package nomos

import (
	"github.com/google/nomos/cmd/nomos/initialize"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(initCmd)
}

var initCmd = &cobra.Command{
	Use:   "init DIRECTORY",
	Short: "Initialize a GKE Policy Management directory",
	Long: `Initialize a GKE Policy Management directory

Given an empty directory, sets up a working GKE Policy Management directory.
Returns an error if the given directory is non-empty.`,
	Example: `  nomos init
  nomos init --path=my/directory
  nomos init --path=/path/to/my/directory`,
	Args: cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		err := initialize.Initialize(nomosPath)
		if err != nil {
			printErrAndDie(err)
		}
	},
}
