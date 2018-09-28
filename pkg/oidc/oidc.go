package oidc

import (
	"flag"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// rootCmd is the top-level command for the plugin.
var rootCmd = &cobra.Command{
	Use:   "oidc",
	Short: "Short description",
	Long:  "Long description",
	Run: func(cmd *cobra.Command, args []string) {
	},
}

var kubeConfig string

func init() {
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)

	rootCmd.PersistentFlags().StringVar(&kubeConfig,
		"kube-config", os.Getenv("KUBECTL_PLUGINS_LOCAL_FLAG_KUBE_CONFIG"),
		"the path to a Kubernetes configuration file.  If undefined, defaults are used.")
}

// Execute is the entry point to the OIDC comand tree.
func Execute() {
	if len(os.Args) < 2 {
		fmt.Println(rootCmd.UsageString())
		os.Exit(1)
	}
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
