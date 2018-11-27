package nomos

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "nomos",
	Short: "Set up and manage a GKE Policy Management directory",
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&validate, "validate", true,
		`If true, use a schema to validate the GKE Policy Management directory.
`)
	rootCmd.PersistentFlags().Var(&nomosPath, "path",
		`The path to use as a GKE Policy Management directory. Defaults to the working directory.
`)
}

var validate bool
var nomosPath = WorkingDirectoryPath

// Execute executes the root nomos command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
