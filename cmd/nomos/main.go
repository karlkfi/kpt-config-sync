package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/google/nomos/cmd/nomos/hydrate"
	"github.com/google/nomos/cmd/nomos/initialize"
	"github.com/google/nomos/cmd/nomos/status"
	"github.com/google/nomos/cmd/nomos/version"
	"github.com/google/nomos/cmd/nomos/vet"
	"github.com/google/nomos/pkg/api/configmanagement"
	pkgversion "github.com/google/nomos/pkg/version"
	"github.com/spf13/cobra"
)

var (
	rootCmd = &cobra.Command{
		Use: configmanagement.CLIName,
		Short: fmt.Sprintf(
			"Set up and manage a Anthos Configuration Management directory (version %v)", pkgversion.VERSION),
	}
)

func init() {
	rootCmd.AddCommand(initialize.Cmd)
	rootCmd.AddCommand(hydrate.Cmd)
	rootCmd.AddCommand(vet.Cmd)
	rootCmd.AddCommand(version.Cmd)
	rootCmd.AddCommand(status.Cmd)
}

func init() {
	// TODO: Re-enable --contexts as a global flag once all subcommands handle it.
	//	pf := rootCmd.PersistentFlags()
	//	pf.StringSliceVar(&flags.Contexts, flags.ContextsName, nil,
	//		`Accepts a comma-separated list of contexts to use in multi-cluster commands. Defaults to all contexts. Use "" for no contexts.
	//`)
	//	_ = pf.MarkHidden(flags.ContextsName)
}

func main() {
	// glog gripes if you don't parse flags before making any logging statements.
	flag.CommandLine.Parse([]string{}) // nolint:errcheck
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
