package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/google/nomos/cmd/nomos/flags"
	"github.com/google/nomos/cmd/nomos/importer"
	"github.com/google/nomos/cmd/nomos/initialize"
	"github.com/google/nomos/cmd/nomos/version"
	"github.com/google/nomos/cmd/nomos/vet"
	"github.com/google/nomos/cmd/nomos/view"
	"github.com/google/nomos/pkg/api/configmanagement"
	pkgversion "github.com/google/nomos/pkg/version"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// logging returns whether to add the logging flags to the binary.
func logging() bool {
	// Set to true to enable logging for internal developer use.
	// Do not check in or release to customers if set to true.
	return false
}

var (
	rootCmd = &cobra.Command{
		Use: configmanagement.CLIName,
		Short: fmt.Sprintf(
			"Set up and manage a CSP Configuration Management directory (version %v)", pkgversion.VERSION),
	}
)

func init() {
	rootCmd.AddCommand(initialize.InitCmd)
	rootCmd.AddCommand(vet.VetCmd)
	rootCmd.AddCommand(view.PrintCmd)
	rootCmd.AddCommand(importer.Cmd)
	rootCmd.AddCommand(version.Cmd)
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&flags.Validate, flags.ValidateFlag, true,
		`If true, use a schema to validate the CSP Configuration Management directory.
`)
	rootCmd.PersistentFlags().Var(&flags.Path, flags.PathFlag,
		`The path to use as a CSP Configuration Management directory. Defaults to the working directory.
`)
}

func main() {
	if logging() {
		pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	}
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
