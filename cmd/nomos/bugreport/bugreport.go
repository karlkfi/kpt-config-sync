package bugreport

import (
	"flag"
	"fmt"

	"github.com/golang/glog"
	"github.com/google/nomos/pkg/api/configmanagement"
	"github.com/google/nomos/pkg/bugreport"
	"github.com/google/nomos/pkg/client/restconfig"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// Cmd retrieves readers for all relevant nomos container logs and cluster state commands and writes them to a zip file
var Cmd = &cobra.Command{
	Use:   "bugreport",
	Short: fmt.Sprintf("Generates a zip file of relevant %v debug information.", configmanagement.CLIName),
	Long:  "Generates a zip file in your current directory containing an aggregate of the logs and cluster state for debugging purposes.",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Don't show usage on error, as argument validation passed.
		cmd.SilenceUsage = true

		// hack to set the hidden variable in glog to also print info statements
		// cobra does not expose core golang-style flags
		if err := flag.CommandLine.Parse([]string{"--stderrthreshold=0"}); err != nil {
			glog.Errorf("could not increase logging verbosity: %v", err)
		}

		cfg, err := restconfig.NewRestConfig(restconfig.DefaultTimeout)
		if err != nil {
			return errors.Wrapf(err, "failed to create rest config")
		}

		report, err := bugreport.New(cmd.Context(), cfg)
		if err != nil {
			return errors.Wrap(err, "failed to initialize bug reporter")
		}

		if err = report.Open(); err != nil {
			return err
		}

		report.WriteRawInZip(report.FetchLogSources(cmd.Context()))
		report.WriteRawInZip(report.FetchResources(cmd.Context()))
		report.WriteRawInZip(report.FetchCMSystemPods(cmd.Context()))
		report.AddNomosStatusToZip(cmd.Context())
		report.AddNomosVersionToZip(cmd.Context())

		report.Close()
		return nil
	},
}
