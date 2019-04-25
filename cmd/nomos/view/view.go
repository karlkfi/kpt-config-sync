package view

import (
	"encoding/json"
	"fmt"

	"github.com/google/nomos/cmd/nomos/flags"
	"github.com/google/nomos/cmd/nomos/parse"
	"github.com/google/nomos/cmd/nomos/util"
	"github.com/google/nomos/pkg/importer/filesystem"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func init() {
	flags.AddPath(Cmd)
	flags.AddValidate(Cmd)
}

// Cmd is the Cobra object representing the nomos view command.
var Cmd = &cobra.Command{
	Use:   "view",
	Short: "View compiled objects from a CSP Configuration Management directory",
	Long: `View compiled objects from a CSP Configuration Management directory
Parses a CSP Configuration Management directory and prints a representation of the
objects it contains.
If errors are encountered during parsing, prints those errors and returns a
non-zero error code.`,
	Example: `  nomos view
  nomos view --path=my/directory
  nomos view --path=/path/to/my/directory`,
	Args: cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		dir := flags.Path.String()
		resources, err := parse.Parse(
			dir,
			filesystem.ParserOpt{Validate: flags.Validate, Vet: true, Extension: &filesystem.NomosVisitorProvider{}})
		if err != nil {
			util.PrintErrAndDie(err)
		}
		err = prettyPrint(resources)
		if err != nil {
			util.PrintErrAndDie(errors.Wrap(err, "Failed to print generated CRDs"))
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
