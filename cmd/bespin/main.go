package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/google/nomos/cmd/bespin/compile"
	"github.com/google/nomos/cmd/nomos/flags"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	rootCmd = &cobra.Command{
		Use:   "bespin",
		Short: "Bespin related support commands",
	}
)

func init() {
	rootCmd.AddCommand(compile.Cmd)
}

func init() {
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
