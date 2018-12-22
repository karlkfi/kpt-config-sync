package view

import (
	"os"
	"path/filepath"
	"time"

	"github.com/golang/glog"
	"github.com/google/nomos/cmd/nomos/flags"
	"github.com/google/nomos/cmd/nomos/parse"
	"github.com/google/nomos/cmd/nomos/util"
	"github.com/google/nomos/pkg/policyimporter/filesystem"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// PrintCmd is the Cobra object representing the nomos view command.
var PrintCmd = &cobra.Command{
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
		dir, err := filepath.Abs(flags.Path.String())
		if err != nil {
			util.PrintErrAndDie(errors.Wrap(err, "Failed to get absolute path"))
		}
		// Check for a set environment variable instead of using a flag so as not to expose
		// this WIP externally.
		e := &parse.Ext{}
		if _, ok := os.LookupEnv("NOMOS_ENABLE_BESPIN"); ok {
			e = &parse.Ext{VP: filesystem.BespinVisitors, Syncs: filesystem.BespinSyncs}
		}
		start := time.Now()
		resources, err := parse.Parse(dir, filesystem.ParserOpt{Validate: flags.Validate, Vet: true, Extension: e})
		if err != nil {
			util.PrintErrAndDie(err)
		}
		glog.Infof("parse took %s", time.Now().Sub(start))

		var handler ResourceEmitter
		if viewPath == "" {
			handler = NewStdoutHandler()
		} else {
			handler = NewFilesystemHandler(viewPath, force)
		}

		if err := handler.Emit(normalizeResources(resources)); err != nil {
			util.PrintErrAndDie(errors.Wrap(err, "Failed to output generated resources"))
		}
	},
}

// viewPath is the path to a directory that the view will be displayed in.
var viewPath string

// force will
var force bool

func init() {
	PrintCmd.Flags().StringVar(
		&viewPath,
		"viewpath",
		"",
		"If defined writes the view out to the filesystem at this path instead of stdout",
	)
	PrintCmd.Flags().BoolVarP(
		&force,
		"force",
		"f",
		false,
		"If viewpath is set, this will recursively remove the directory prior to writing out to the filesystem."+
			"  WARNING: this is equivalent to running rm -rf on the viewpath location, proceed at your own risk!",
	)
}
