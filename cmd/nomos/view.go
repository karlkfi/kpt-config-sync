package nomos

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/nomos/pkg/policyimporter/filesystem"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(printCmd)
}

var printCmd = &cobra.Command{
	Use:   "view",
	Short: "View compiled objects from a GKE Policy Management directory",
	Long: `View compiled objects from a GKE Policy Management directory

Parses a GKE Policy Management directory and prints a representation of the
objects it contains.

If errors are encountered during parsing, prints those errors and returns a
non-zero error code.`,
	Example: `  nomos view
  nomos view --path=my/directory
  nomos view --path=/path/to/my/directory`,
	Args: cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		dir, err := filepath.Abs(nomosPath.Path())
		if err != nil {
			printErrAndDie(errors.Wrap(err, "Failed to get absolute path"))
		}

		// Check for a set environment variable instead of using a flag so as not to expose
		// this WIP externally.
		_, bespin := os.LookupEnv("NOMOS_ENABLE_BESPIN")
		resources := parse(dir, filesystem.ParserOpt{Validate: validate, Vet: true, Bespin: bespin})

		err = prettyPrint(resources)
		if err != nil {
			printErrAndDie(errors.Wrap(err, "Failed to print generated CRDs"))
		}
	},
}

func prettyPrint(v interface{}) (err error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err == nil {
		fmt.Println(string(b))
	}
	return err
}
