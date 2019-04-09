package status

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"text/tabwriter"
	"time"

	"github.com/golang/glog"
	"github.com/google/nomos/clientgen/apis"
	typedv1 "github.com/google/nomos/clientgen/apis/typed/configmanagement/v1"
	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/client/restconfig"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/util/slice"
)

const (
	maxNameLength   = 50
	pollingInterval = time.Second
)

var writer = tabwriter.NewWriter(os.Stdout, 0, 0, 5, ' ', 0)

// Cmd exports resources in the current kubectl context into the specified directory.
var Cmd = &cobra.Command{
	Hidden: true,
	Use:    "status",
	// TODO: make Configuration Management a constant (for product renaming)
	Short: `Prints the status of all clusters with Configuration Management installed.`,
	Run: func(cmd *cobra.Command, args []string) {
		clientMap, err := repoClients()
		if err != nil {
			glog.Fatalf("Failed to get client: %v", err)
		}
		// Use a sorted order of names to avoid shuffling in the output.
		names := clusterNames(clientMap)

		for {
			<-time.Tick(pollingInterval)
			writeMap := make(map[string]string)
			errorMap := make(map[string][]string)

			// First build up maps of all the things we want to display.
			for name, repoClient := range clientMap {
				repoList, listErr := repoClient.List(metav1.ListOptions{})
				if listErr != nil {
					writeMap[name] = errorRow(name, "Config Management is not configured")
					continue
				}

				if len(repoList.Items) > 0 {
					repoStatus := repoList.Items[0].Status
					writeMap[name] = statusRow(name, repoStatus)

					errs := statusErrors(repoStatus)
					if len(errs) > 0 {
						errorMap[name] = errs
					}
				} else {
					writeMap[name] = errorRow(name, "Cluster status is unavailable")
				}
			}

			// Now we write everything at once. Processing and then printing helps avoid screen strobe.

			// Clear previous input and flush to avoid it messing up column widths.
			clearTerminal()
			if err = writer.Flush(); err != nil {
				glog.Warningf("Failed to clear terminal: %v", err)
			}

			// Print table header.
			_, err = fmt.Fprintf(writer, "%s\tStatus\tCurrent Config\t\n", "Cluster")
			if err != nil {
				glog.Warningf("Failed to print header: %v", err)
			}

			// Print a summary of all clusters.
			for _, name := range names {
				_, err = fmt.Fprintf(writer, "%s", writeMap[name])
				if err != nil {
					glog.Warningf("Failed to print cluster status: %v", err)
				}
			}

			// Print out any errors that occurred.
			if len(errorMap) > 0 {
				_, err = fmt.Fprintf(writer, "\n\n")
				if err != nil {
					glog.Warningf("Failed to print line break: %v", err)
				}
				_, err = fmt.Fprintln(writer, "Config Management Errors:")
				if err != nil {
					glog.Warningf("Failed to print error header: %v", err)
				}

				for _, name := range names {
					for _, clusterErr := range errorMap[name] {
						_, err = fmt.Fprintln(writer, errorRow(name, clusterErr))
						if err != nil {
							glog.Warningf("Failed to print cluster error: %v", err)
						}
					}
				}
			}

			if err = writer.Flush(); err != nil {
				glog.Warningf("Failed to flush: %v", err)
			}
		}
	},
}

func repoClients() (map[string]typedv1.RepoInterface, error) {
	configs, err := restconfig.AllKubectlConfigs()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create client configs")
	}

	clientMap := make(map[string]typedv1.RepoInterface, len(configs))

	for name, cfg := range configs {
		policyHierarchyClientSet, err := apis.NewForConfig(cfg)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to create configmanagement clientset")
		}
		clientMap[name] = policyHierarchyClientSet.ConfigmanagementV1().Repos()
	}

	return clientMap, nil
}

func clusterNames(clientMap map[string]typedv1.RepoInterface) []string {
	var names []string
	for name := range clientMap {
		names = append(names, name)
	}
	return slice.SortStrings(names)
}

func clearTerminal() {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "cls")
	default:
		cmd = exec.Command("clear")
	}

	cmd.Stdout = writer
	if err := cmd.Run(); err != nil {
		glog.Warningf("Failed to execute command: %v", err)
	}
}

func shortName(name string) string {
	if len(name) <= maxNameLength {
		return name
	}
	return name[len(name)-maxNameLength:]
}

func errorRow(name string, err string) string {
	return fmt.Sprintf("%s\t%s\n", shortName(name), err)
}

func statusRow(name string, status v1.RepoStatus) string {
	return fmt.Sprintf("%s\t%s\t%s\t\n", shortName(name), getStatus(status), status.Sync.LatestToken)
}

func getStatus(status v1.RepoStatus) string {
	if status.Sync.LatestToken == status.Source.Token {
		if len(status.Sync.InProgress) == 0 {
			return "Configs up-to-date"
		}
		return "Applying configs"
	}
	if status.Import.Token == status.Source.Token {
		if len(status.Import.Errors) == 0 {
			return "Applying configs"
		}
		return "Parsing configs"
	}
	return "Parsing configs"
}

func statusErrors(status v1.RepoStatus) []string {
	var errs []string
	for _, err := range status.Import.Errors {
		errs = append(errs, err.ErrorMessage)
	}
	for _, syncStatus := range status.Sync.InProgress {
		for _, err := range syncStatus.Errors {
			errs = append(errs, err.ErrorMessage)
		}
	}
	return errs
}
