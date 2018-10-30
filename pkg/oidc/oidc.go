package oidc

import (
	"flag"
	"fmt"
	"os"
	"os/user"
	"path/filepath"

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
	PreRunE: func(_ *cobra.Command, _ []string) error {
		if kubeConfig == "" {
			// Try to load the default user kubeconfig file if not specified.
			// This requires cgo so may fail in some statically linked environments.
			user, err := user.Current()
			if err != nil {
				return fmt.Errorf("could not determine current user: %v", err)
			}
			kubeConfig = filepath.Join(user.HomeDir, ".kube", "config")
		}
		if _, err := os.Stat(kubeConfig); err != nil {
			return fmt.Errorf("could not find config file: %q", kubeConfig)
		}
		return nil
	},
}

var kubeConfig string

func init() {
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)

	rootCmd.PersistentFlags().StringVar(&kubeConfig,
		"kube-config", os.Getenv("KUBECONFIG"),
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
