package vet

import (
	"fmt"

	"github.com/google/nomos/cmd/nomos/flags"
	"github.com/google/nomos/pkg/api/configmanagement"
	"github.com/google/nomos/pkg/importer/filesystem"
	"github.com/google/nomos/pkg/reconcilermanager"
	"github.com/spf13/cobra"
)

var (
	sourceFormatValue string

	namespaceFlag  = "namespace"
	namespaceValue string
)

func init() {
	flags.AddClusters(Cmd)
	flags.AddPath(Cmd)
	flags.AddSkipAPIServerCheck(Cmd)
	Cmd.Flags().StringVar(&sourceFormatValue, reconcilermanager.SourceFormat, "",
		fmt.Sprintf("If %q or unset, validate as a %s repository. If %q, validate as an unstructured repository.",
			string(filesystem.SourceFormatHierarchy), configmanagement.ProductName,
			string(filesystem.SourceFormatUnstructured)))
	Cmd.Flags().StringVar(&namespaceValue, namespaceFlag, "",
		fmt.Sprintf(
			"If set, validate the repository as a Namespace Repo with the provided name. Automatically sets --source-format=%s",
			filesystem.SourceFormatUnstructured))
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
	RunE: func(cmd *cobra.Command, args []string) error {
		// Don't show usage on error, as argument validation passed.
		cmd.SilenceUsage = true

		return runVet(flags.Path, namespaceValue, filesystem.SourceFormat(sourceFormatValue),
			flags.SkipAPIServer, flags.AllClusters(), flags.Clusters)
	},
}
