package status

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"runtime"
	"sync"
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
	maxNameLength = 50
)

var (
	clientTimeout   time.Duration
	pollingInterval time.Duration
	writer          = tabwriter.NewWriter(os.Stdout, 0, 0, 5, ' ', 0)
)

func init() {
	Cmd.Flags().DurationVar(&clientTimeout, "timeout", 3*time.Second, "Timeout for connecting to each cluster")
	Cmd.Flags().DurationVar(&pollingInterval, "poll", 0*time.Second, "Polling interval (leave unset to run once)")
}

// Cmd runs a loop that fetches Repos from all available clusters and prints a summary of the status
// of Config Management for each cluster.
var Cmd = &cobra.Command{
	Hidden: true,
	Use:    "status",
	// TODO: make Configuration Management a constant (for product renaming)
	Short: `Prints the status of all clusters with Configuration Management installed.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Connecting to clusters...")

		clientMap, err := repoClients()
		if err != nil {
			glog.Fatalf("Failed to get clients: %v", err)
		}
		// Use a sorted order of names to avoid shuffling in the output.
		names := clusterNames(clientMap)

		if pollingInterval > 0 {
			for {
				printRepos(clientMap, names)
				time.Sleep(pollingInterval)
			}
		} else {
			printRepos(clientMap, names)
		}
	},
}

// repoClients returns a map of of typed clients keyed by the name of the kubeconfig context they
// are initialized from.
func repoClients() (map[string]typedv1.RepoInterface, error) {
	configs, err := restconfig.AllKubectlConfigs(clientTimeout)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create client configs")
	}

	clientMap := make(map[string]typedv1.RepoInterface)
	for name, cfg := range configs {
		policyHierarchyClientSet, err := apis.NewForConfig(cfg)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to create configmanagement clientset")
		}
		// Do a quick ping to see if the cluster is healthy/reachable and filter it out if it is not.
		if isReachable(policyHierarchyClientSet, name) {
			clientMap[name] = policyHierarchyClientSet.ConfigmanagementV1().Repos()
		}
	}

	if len(clientMap) < len(configs) {
		// We can't stop the underlying libraries from spamming to glog when a cluster is unreachable,
		// so just flush it out and print a blank line to at least make a clean separation.
		glog.Flush()
		fmt.Println()
	}
	return clientMap, nil
}

// isReachable returns true if the given ClientSet points to a reachable cluster.
func isReachable(clientset *apis.Clientset, cluster string) bool {
	_, err := clientset.RESTClient().Get().DoRaw()
	if err == nil {
		return true
	}
	if nErr, ok := err.(net.Error); ok && nErr.Timeout() {
		fmt.Printf("%q is an invalid cluster\n", cluster)
	} else {
		fmt.Printf("Failed to connect to cluster %q: %v\n", cluster, err)
	}
	return false
}

// clusterNames returns a sorted list of names from the given clientMap.
func clusterNames(clientMap map[string]typedv1.RepoInterface) []string {
	var names []string
	for name := range clientMap {
		names = append(names, name)
	}
	return slice.SortStrings(names)
}

// printRepos fetches RepoStatus from each cluster in the given map and then prints a formatted
// status row for each one. If there are any errors reported by the RepoStatus, those are printed in
// a second table under the status table.
// nolint:errcheck
func printRepos(clientMap map[string]typedv1.RepoInterface, names []string) {
	// First build up maps of all the things we want to display.
	writeMap, errorMap := fetchRepos(clientMap)
	// Now we write everything at once. Processing and then printing helps avoid screen strobe.

	if pollingInterval > 0 {
		// Clear previous output and flush it to avoid messing up column widths.
		clearTerminal()
		writer.Flush()
	}

	// Print table header.
	fmt.Fprintln(writer, "Cluster\tStatus\tCurrent Config\t")
	fmt.Fprintln(writer, "-------\t------\t--------------\t")

	// Print a summary of all clusters.
	for _, name := range names {
		fmt.Fprintf(writer, "%s", writeMap[name])
	}

	// Print out any errors that occurred.
	if len(errorMap) > 0 {
		fmt.Fprintf(writer, "\n\n")
		fmt.Fprintln(writer, "Config Management Errors:")

		for _, name := range names {
			for _, clusterErr := range errorMap[name] {
				fmt.Fprintln(writer, errorRow(name, clusterErr))
			}
		}
	}

	writer.Flush()
}

// clusterStatus specifies the name, status, and any config management errors for a cluster.
type clusterStatus struct {
	name   string
	status string
	errs   []string
}

// fetchRepos returns two maps which are both keyed by cluster name. The first is a map of printable
// cluster status rows and the second is a map of printable cluster error rows.
func fetchRepos(clientMap map[string]typedv1.RepoInterface) (map[string]string, map[string][]string) {
	writeMap := make(map[string]string)
	errorMap := make(map[string][]string)
	statusCh := make(chan clusterStatus)
	wg := sync.WaitGroup{}

	// We fetch the repo objects in parallel to avoid long delays if multiple clusters are unreachable
	// or slow to respond.
	for name, repoClient := range clientMap {
		wg.Add(1)

		go func(name string, repoClient typedv1.RepoInterface) {
			result := clusterStatus{name: name}
			repoList, listErr := repoClient.List(metav1.ListOptions{})

			if listErr != nil {
				result.status = errorRow(name, "Config Management is not installed")
			} else if len(repoList.Items) == 0 {
				result.status = errorRow(name, "Cluster status is unavailable")
			} else {
				repoStatus := repoList.Items[0].Status
				result.status = statusRow(name, repoStatus)
				result.errs = statusErrors(repoStatus)
			}

			statusCh <- result
			wg.Done()
		}(name, repoClient)
	}

	go func() {
		wg.Wait()
		close(statusCh)
	}()

	for result := range statusCh {
		writeMap[result.name] = result.status
		if len(result.errs) > 0 {
			errorMap[result.name] = result.errs
		}
	}

	return writeMap, errorMap
}

// clearTerminal executes an OS-specific command to clear all output on the terminal.
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

// shortName returns a cluster name which has been truncated to the maximum name length.
func shortName(name string) string {
	if len(name) <= maxNameLength {
		return name
	}
	return name[len(name)-maxNameLength:]
}

// errorRow returns the given error message formated as a printable row.
func errorRow(name string, err string) string {
	return fmt.Sprintf("%s\t%s\n", shortName(name), err)
}

// statusRow returns the given RepoStatus formated as a printable row.
func statusRow(name string, status v1.RepoStatus) string {
	token := status.Sync.LatestToken
	if token == "" {
		token = "N/A"
	}
	return fmt.Sprintf("%s\t%s\t%s\t\n", shortName(name), getStatus(status), token)
}

// getStatus returns the given RepoStatus formatted as a short summary string.
func getStatus(status v1.RepoStatus) string {
	if hasErrors(status) {
		return "Error (details below)"
	}
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

// hasErrors returns true if there are any config management errors present in the given RepoStatus.
func hasErrors(status v1.RepoStatus) bool {
	if len(status.Import.Errors) > 0 {
		return true
	}
	for _, syncStatus := range status.Sync.InProgress {
		if len(syncStatus.Errors) > 0 {
			return true
		}
	}
	return false
}

// statusErrors returns all errors reported in the given RepoStatus as a single array.
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
