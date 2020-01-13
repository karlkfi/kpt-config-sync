package vet

import (
	"fmt"
	"os"

	"github.com/google/nomos/cmd/nomos/flags"
	"github.com/google/nomos/cmd/nomos/parse"
	"github.com/google/nomos/cmd/nomos/util"
	"github.com/google/nomos/pkg/api/configmanagement"
	"github.com/google/nomos/pkg/hydrate"
	"github.com/google/nomos/pkg/importer"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/filesystem"
	"github.com/google/nomos/pkg/status"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	// TODO(b/147098697): Make identical to sourceFormat directive.
	disableHierarchyFlag = "disable-hierarchy"
	disableHierarchy     bool
)

func init() {
	flags.AddClusters(Cmd)
	flags.AddPath(Cmd)
	flags.AddSkipAPIServerCheck(Cmd)
	Cmd.Flags().BoolVar(&disableHierarchy, disableHierarchyFlag, false,
		fmt.Sprintf("If true, validate as a %s Repo.\n"+
			"If false, validate recursively as a directory of manifests.", configmanagement.ProductName))
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
	Run: func(cmd *cobra.Command, args []string) {
		rootDir := flags.Path.String()
		rootPath := util.GetRootOrDie(rootDir)

		var parser filesystem.ConfigParser
		if disableHierarchy {
			parser = filesystem.NewRawParser(rootPath, &filesystem.FileReader{}, importer.DefaultCLIOptions)
		} else {
			parser = parse.NewParser(rootPath)
		}

		encounteredError := false
		hydrate.ForEachCluster(parser, parse.GetSyncedCRDs, !flags.SkipAPIServer, vetCluster(&encounteredError))

		if encounteredError {
			os.Exit(1)
		}
	},
}

func vetCluster(encounteredError *bool) func(clusterName string, fileObjects []ast.FileObject, errs status.MultiError) {
	return func(clusterName string, _ []ast.FileObject, errs status.MultiError) {
		clusterEnabled := flags.AllClusters()
		for _, cluster := range flags.Clusters {
			if clusterName == cluster {
				clusterEnabled = true
			}
		}
		if !clusterEnabled {
			return
		}

		if errs != nil {
			if len(errs.Errors()) == 1 && errs.Errors()[0].Code() == status.APIServerErrorCode {
				util.PrintErrOrDie(errors.Wrapf(errs, "did you mean to run with --%s?", flags.SkipAPIServerFlag))
				return
			}

			if clusterName == "" {
				clusterName = parse.UnregisteredCluster
			}
			util.PrintErrOrDie(errors.Wrapf(errs, "errors for Cluster %q", clusterName))
			*encounteredError = true
		}
	}
}
