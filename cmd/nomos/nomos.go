package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/google/nomos/cmd/nomos/bugreport"
	"github.com/google/nomos/cmd/nomos/hydrate"
	"github.com/google/nomos/cmd/nomos/initialize"
	"github.com/google/nomos/cmd/nomos/migrate"
	"github.com/google/nomos/cmd/nomos/status"
	"github.com/google/nomos/cmd/nomos/version"
	"github.com/google/nomos/cmd/nomos/vet"
	"github.com/google/nomos/pkg/api/configmanagement"
	pkgversion "github.com/google/nomos/pkg/version"
	"github.com/spf13/cobra"
	"k8s.io/klog/v2"
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
	rootCmd.AddCommand(bugreport.Cmd)
	rootCmd.AddCommand(migrate.Cmd)
}

func main() {
	klog.InitFlags(nil)
	flag.Parse()

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
