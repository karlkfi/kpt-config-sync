package bugreport

import (
	"context"
	"flag"
	"fmt"

	"github.com/golang/glog"
	"github.com/google/nomos/pkg/api/configmanagement"
	"github.com/google/nomos/pkg/bugreport"
	"github.com/google/nomos/pkg/client/restconfig"
	"github.com/spf13/cobra"
)

// Cmd retrieves readers for all relevant nomos container logs and cluster state commands and writes them to a zip file
var Cmd = &cobra.Command{
	Use:   "bugreport",
	Short: fmt.Sprintf("Generates a zip file of relevant %v debug information.", configmanagement.CLIName),
	Long:  "Generates a zip file in your current directory containing an aggregate of the logs and cluster state for debugging purposes.",
	Run: func(cmd *cobra.Command, args []string) {
		// hack to set the hidden variable in glog to also print info statements
		// cobra does not expose core golang-style flags
		if err := flag.CommandLine.Parse([]string{"--stderrthreshold=0"}); err != nil {
			glog.Errorf("could not increase logging verbosity: %v", err)
		}

		cfg, err := restconfig.NewRestConfig()
		if err != nil {
			glog.Fatalf("failed to create rest config: %v", err)
		}

		report, err := bugreport.New(context.Background(), cfg)
		if err != nil {
			glog.Fatalf("failed to initialize bug reporter: %v", err)
		}

		if err = report.Open(); err != nil {
			glog.Fatal(err)
		}

		report.WriteRawInZip(report.FetchLogSources())
		report.WriteRawInZip(report.FetchCMResources())
		report.WriteRawInZip(report.FetchCMSystemPods())
		report.AddNomosStatusToZip(cmd.Context())
		report.AddNomosVersionToZip()

		report.Close()
	},
}
