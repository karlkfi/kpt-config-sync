package vet

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/nomos/cmd/nomos/flags"
	"github.com/google/nomos/cmd/nomos/parse"
	"github.com/google/nomos/cmd/nomos/util"
	"github.com/google/nomos/pkg/api/configmanagement"
	"github.com/google/nomos/pkg/hydrate"
	"github.com/google/nomos/pkg/importer"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/filesystem"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/status"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	hierarchyFormat    = "hierarchy"
	unstructuredFormat = "unstructured"
)

var (
	sourceFormatFlag = "source-format"
	sourceFormat     string
)

func init() {
	flags.AddClusters(Cmd)
	flags.AddPath(Cmd)
	flags.AddSkipAPIServerCheck(Cmd)
	Cmd.Flags().StringVar(&sourceFormat, sourceFormatFlag, hierarchyFormat,
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
		abs, err := filepath.Abs(flags.Path.String())
		if err != nil {
			util.PrintErrAndDie(err)
		}
		rootDir, err := cmpath.AbsoluteOS(abs)
		if err != nil {
			util.PrintErrAndDie(err)
		}
		rootDir, err = rootDir.EvalSymlinks()
		if err != nil {
			util.PrintErrAndDie(err)
		}

		files, err := parse.FindFiles(rootDir)
		if err != nil {
			util.PrintErrAndDie(err)
		}
		var parser filesystem.ConfigParser
		switch sourceFormat {
		case hierarchyFormat:
			parser = parse.NewParser()
			files = filesystem.FilterHierarchyFiles(rootDir, files)
		case unstructuredFormat:
			parser = filesystem.NewRawParser(&filesystem.FileReader{}, importer.DefaultCLIOptions)
		default:
			return fmt.Errorf("unknown %s value %q", sourceFormatFlag, sourceFormat)
		}

		encounteredError := false
		hydrate.ForEachCluster(parser, parse.GetSyncedCRDs, !flags.SkipAPIServer, rootDir, files, vetCluster(&encounteredError))

		if encounteredError {
			os.Exit(1)
		}
		return nil
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
