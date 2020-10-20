package status

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"text/tabwriter"
	"time"

	"github.com/golang/glog"
	"github.com/google/nomos/cmd/nomos/flags"
	"github.com/google/nomos/cmd/nomos/util"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/client/restconfig"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	pendingMsg = "PENDING"
	syncedMsg  = "SYNCED"
)

var (
	clientTimeout   time.Duration
	pollingInterval time.Duration
)

func init() {
	flags.AddContexts(Cmd)
	Cmd.Flags().DurationVar(&clientTimeout, "timeout", 3*time.Second, "Timeout for connecting to each cluster")
	Cmd.Flags().DurationVar(&pollingInterval, "poll", 0*time.Second, "Polling interval (leave unset to run once)")
}

// GetStatusReadCloser returns a ReadCloser with the output produced by running the "nomos status" command as a string
func GetStatusReadCloser(ctx context.Context, contexts []string) (io.ReadCloser, error) {
	r, w, _ := os.Pipe()
	writer := util.NewWriter(w)

	clientMap, err := statusClients(contexts)
	if err != nil {
		return nil, err
	}
	names := clusterNames(clientMap)

	printStatus(ctx, writer, clientMap, names)
	err = w.Close()
	if err != nil {
		e := fmt.Errorf("failed to close status file writer with error: %v", err)
		return nil, e
	}

	return ioutil.NopCloser(r), nil
}

// Cmd runs a loop that fetches ACM objects from all available clusters and prints a summary of the
// status of Config Management for each cluster.
var Cmd = &cobra.Command{
	Use: "status",
	// TODO: make Configuration Management a constant (for product renaming)
	Short: `Prints the status of all clusters with Configuration Management installed.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Connecting to clusters...")

		clientMap, err := statusClients(flags.Contexts)
		if err != nil {
			// If "no such file or directory" error, unwrap and display before exiting
			if unWrapped := errors.Cause(err); os.IsNotExist(unWrapped) {
				// nolint:errcheck
				fmt.Printf("failed to create client configs: %v\n", unWrapped)
				os.Exit(255)
			}

			glog.Fatalf("Failed to get clients: %v", err)
		}
		if len(clientMap) == 0 {
			fmt.Print("No clusters found.\n")
			os.Exit(255)
		}

		// Use a sorted order of names to avoid shuffling in the output.
		names := clusterNames(clientMap)

		writer := util.NewWriter(os.Stdout)
		if pollingInterval > 0 {
			for {
				printStatus(cmd.Context(), writer, clientMap, names)
				time.Sleep(pollingInterval)
			}
		} else {
			printStatus(cmd.Context(), writer, clientMap, names)
		}
	},
}

// clusterNames returns a sorted list of names from the given clientMap.
func clusterNames(clientMap map[string]*statusClient) []string {
	var names []string
	for name := range clientMap {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// clusterStates returns a map of clusterStates calculated from the given map of clients.
func clusterStates(ctx context.Context, clientMap map[string]*statusClient) map[string]*clusterState {
	stateMap := make(map[string]*clusterState)
	for name, client := range clientMap {
		if client == nil {
			stateMap[name] = unavailableCluster(name)
		} else {
			stateMap[name] = client.clusterStatus(ctx, name)
		}
	}
	return stateMap
}

// printStatus fetches ConfigManagementStatus and/or RepoStatus from each cluster in the given map
// and then prints a formatted status row for each one. If there are any errors reported by either
// object, those are printed in a second table under the status table.
// nolint:errcheck
func printStatus(ctx context.Context, writer *tabwriter.Writer, clientMap map[string]*statusClient, names []string) {
	// First build up a map of all the states to display.
	stateMap := clusterStates(ctx, clientMap)

	currentContext, err := restconfig.CurrentContextName()
	if err != nil {
		fmt.Printf("Failed to get current context name with err: %v\n", errors.Cause(err))
	}

	// Now we write everything at once. Processing and then printing helps avoid screen strobe.

	if pollingInterval > 0 {
		// Clear previous output and flush it to avoid messing up column widths.
		clearTerminal(writer)
		writer.Flush()
	}

	// Print status for each cluster.
	for _, name := range names {
		state := stateMap[name]
		if name == currentContext {
			// Prepend an asterisk for the users' current context
			state.ref = "*" + name
		}
		state.printRows(writer)
	}

	writer.Flush()
}

// clearTerminal executes an OS-specific command to clear all output on the terminal.
func clearTerminal(out io.Writer) {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "cls")
	default:
		cmd = exec.Command("clear")
	}

	cmd.Stdout = out
	if err := cmd.Run(); err != nil {
		glog.Warningf("Failed to execute command: %v", err)
	}
}

func getResourceStatus(resourceConditions []v1.ResourceCondition) v1.ResourceConditionState {
	resourceStatus := v1.ResourceStateHealthy

	for _, resourceCondition := range resourceConditions {

		if resourceCondition.ResourceState.IsError() {
			return v1.ResourceStateError
		} else if resourceCondition.ResourceState.IsReconciling() {
			resourceStatus = v1.ResourceStateReconciling
		}
	}

	return resourceStatus
}

func getResourceStatusErrors(resourceConditions []v1.ResourceCondition) []string {
	if len(resourceConditions) == 0 {
		return nil
	}

	var syncErrors []string

	for _, resourceCondition := range resourceConditions {
		for _, rcError := range resourceCondition.Errors {
			syncErrors = append(syncErrors, fmt.Sprintf("%v\t%v\tError: %v", resourceCondition.Kind, resourceCondition.NamespacedName, rcError))
		}
		for _, rcReconciling := range resourceCondition.ReconcilingReasons {
			syncErrors = append(syncErrors, fmt.Sprintf("%v\t%v\tReconciling: %v", resourceCondition.Kind, resourceCondition.NamespacedName, rcReconciling))
		}
	}

	return syncErrors
}

// getSyncStatus returns the given RepoStatus formatted as a short summary string.
func getSyncStatus(status v1.RepoStatus) string {
	if hasErrors(status) {
		return util.ErrorMsg
	}
	if len(status.Sync.LatestToken) == 0 {
		return pendingMsg
	}
	if status.Sync.LatestToken == status.Source.Token && len(status.Sync.InProgress) == 0 {
		return syncedMsg
	}
	return pendingMsg
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

// syncStatusErrors returns all errors reported in the given RepoStatus as a single array.
func syncStatusErrors(status v1.RepoStatus) []string {
	var errs []string
	for _, err := range status.Source.Errors {
		errs = append(errs, err.ErrorMessage)
	}
	for _, err := range status.Import.Errors {
		errs = append(errs, err.ErrorMessage)
	}
	for _, syncStatus := range status.Sync.InProgress {
		for _, err := range syncStatus.Errors {
			errs = append(errs, err.ErrorMessage)
		}
	}

	if getResourceStatus(status.Sync.ResourceConditions) != v1.ResourceStateHealthy {
		errs = append(errs, getResourceStatusErrors(status.Sync.ResourceConditions)...)
	}

	return errs
}
