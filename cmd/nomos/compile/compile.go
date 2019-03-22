package compile

import (
	"time"

	"github.com/golang/glog"
	"github.com/google/nomos/cmd/nomos/flags"
	"github.com/google/nomos/cmd/nomos/parse"
	"github.com/google/nomos/cmd/nomos/util"
	"github.com/google/nomos/pkg/importer/filesystem"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// Cmd is the Cobra object representing the nomos view command.
var Cmd = &cobra.Command{
	Use:   "compile",
	Short: "Compile objects from a CSP Configuration Management directory",
	Long: `Output compiled objects from a CSP Configuration Management directory.  This
parses the directory then outputs YAML as it will be applied to the API server
sans any implementation specific Custom Resources involved.  If errors are encountered
during parsing, prints those errors and returns a non-zero error code.`,
	Example: `  nomos compile
  nomos compile --path=my/directory
  nomos compile --path=/path/to/my/directory`,
	Args: cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		dir := flags.Path.String()
		start := time.Now()
		resources, err := parse.Parse(
			dir,
			filesystem.ParserOpt{Validate: flags.Validate, Vet: true, Extension: &filesystem.NomosVisitorProvider{}})
		if err != nil {
			util.PrintErrAndDie(err)
		}
		glog.Infof("parse took %s", time.Since(start))

		var handler ResourceEmitter
		if output == "" {
			handler = NewStdoutHandler()
		} else {
			handler = NewFilesystemHandler(output, force)
		}

		if err := handler.Emit(normalizeResources(resources)); err != nil {
			util.PrintErrAndDie(errors.Wrap(err, "Failed to output generated resources"))
		}
	},
}

var (
	// output is the path to a directory that the view will be displayed in.
	output string

	// force will output to an existing non-empty directory by removing the contents.
	force bool
)

func init() {
	Cmd.Flags().StringVar(
		&output,
		"output",
		"",
		"If defined writes the compiled output to the filesystem at this path instead of stdout",
	)
	Cmd.Flags().BoolVar(
		&force,
		"force",
		false,
		"If output is set, this will recursively remove the directory prior to writing out to the filesystem."+
			"  WARNING: this is equivalent to running rm -rf on the output location, proceed at your own risk!",
	)
}
