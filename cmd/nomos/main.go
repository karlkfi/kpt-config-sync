package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/google/nomos/cmd/nomos/flags"
	"github.com/google/nomos/cmd/nomos/initialize"
	"github.com/google/nomos/cmd/nomos/vet"
	"github.com/google/nomos/cmd/nomos/view"
	"github.com/google/nomos/pkg/api/policyhierarchy"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	rootCmd = &cobra.Command{
		Use:   policyhierarchy.CLIName,
		Short: "Set up and manage a GKE Policy Management directory",
	}
)

func init() {
	rootCmd.AddCommand(initialize.InitCmd)
	rootCmd.AddCommand(vet.VetCmd)
	rootCmd.AddCommand(view.PrintCmd)
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&flags.Validate, flags.ValidateFlag, true,
		`If true, use a schema to validate the GKE Policy Management directory.
`)
	rootCmd.PersistentFlags().Var(&flags.Path, flags.PathFlag,
		`The path to use as a GKE Policy Management directory. Defaults to the working directory.
`)
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
