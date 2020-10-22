package vet

import (
	"fmt"

	"github.com/google/nomos/cmd/nomos/flags"
	"github.com/google/nomos/pkg/api/configmanagement"
	"github.com/spf13/cobra"
)

const (
	hierarchyFormat    = "hierarchy"
	unstructuredFormat = "unstructured"
	defaultFormat      = hierarchyFormat
)

var (
	sourceFormatFlag  = "source-format"
	sourceFormatValue string
)

func init() {
	flags.AddClusters(Cmd)
	flags.AddPath(Cmd)
	flags.AddSkipAPIServerCheck(Cmd)
	Cmd.Flags().StringVar(&sourceFormatValue, sourceFormatFlag, defaultFormat,
		fmt.Sprintf("If %q, validate as a %s repository. If %q, validate as an unstructured repository.",
			hierarchyFormat, configmanagement.ProductName, unstructuredFormat))
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
		return vet(flags.Path, sourceFormatValue, flags.SkipAPIServer, flags.AllClusters(), flags.Clusters)
	},
}
