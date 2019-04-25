package flags

import (
	"github.com/google/nomos/cmd/nomos/repo"
	"github.com/spf13/cobra"
)

const (
	// pathFlag is the flag to set the Path of the Nomos directory.
	pathFlag = "path"

	// validateFlag is the flag to set the Validate value
	validateFlag = "validate"

	// ContextsName is the flag name for the Contexts below.
	ContextsName = "contexts"
)

// AddPath adds the --path flag
func AddPath(cmd *cobra.Command) {
	cmd.Flags().Var(&Path, pathFlag,
		`Root directory to use as a CSP Configuration Management repository.`)
}

// AddValidate adds the --validate flag for whether to use schemas for
// validating
func AddValidate(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&Validate, validateFlag, true,
		`If true, use a schema to validate the CSP Configuration Management repository.`)
}

var (
	// Validate determines whether to use a schema to validate the input
	Validate bool

	// Path says where the Nomos directory is
	Path = repo.WorkingDirectoryPath

	// Contexts contains the list of clusters that are targets of cross-cluster
	// commands.
	Contexts []string
)
