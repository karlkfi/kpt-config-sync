package nomos

import (
	"os"
	"path/filepath"

	"github.com/google/nomos/pkg/policyimporter/filesystem"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(vetCmd)
}

var vetCmd = &cobra.Command{
	Use:   "vet",
	Short: "Validate a GKE Policy Management directory",
	Long: `Validate a GKE Policy Management directory

Checks for semantic and syntactic errors in a GKE Policy Management directory
that will interfere with applying resources. Prints found errors to STDERR and
returns a non-zero error code if any issues are found.
`,
	Example: `  nomos vet
  nomos vet --path=my/directory
  nomos vet --path=/path/to/my/directory`,
	Args: cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		dir, err := filepath.Abs(nomosPath.Path())
		if err != nil {
			printErrAndDie(errors.Wrap(err, "Failed to get absolute path"))
		}

		// Check for a set environment variable instead of using a flag so as not to expose
		// this WIP externally.
		_, bespin := os.LookupEnv("NOMOS_ENABLE_BESPIN")
		parse(dir, filesystem.ParserOpt{Validate: validate, Vet: true, Bespin: bespin})
	},
}
