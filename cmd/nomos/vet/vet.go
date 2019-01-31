package vet

import (
	"path/filepath"

	"github.com/google/nomos/cmd/nomos/flags"
	"github.com/google/nomos/cmd/nomos/parse"
	"github.com/google/nomos/cmd/nomos/util"
	"github.com/google/nomos/pkg/policyimporter/filesystem"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	// ValidateFlag is the value used to set and retrieve the validate flag
	ValidateFlag = "validate"
)

// VetCmd is the Cobra object representing the nomos vet command.
var VetCmd = &cobra.Command{
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
		dir, err := filepath.Abs(flags.Path.String())
		if err != nil {
			util.PrintErrAndDie(errors.Wrap(err, "Failed to get absolute path"))
		}
		_, err = parse.Parse(
			dir,
			filesystem.ParserOpt{Validate: flags.Validate, Vet: true, Extension: &filesystem.NomosVisitorProvider{}})
		if err != nil {
			util.PrintErrAndDie(err)
		}
	},
}
