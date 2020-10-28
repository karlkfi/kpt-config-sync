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
	namespace       string
)

func init() {
	flags.AddContexts(Cmd)
	Cmd.Flags().DurationVar(&clientTimeout, "timeout", 3*time.Second, "Timeout for connecting to each cluster")
	Cmd.Flags().DurationVar(&pollingInterval, "poll", 0*time.Second, "Polling interval (leave unset to run once)")
	Cmd.Flags().StringVar(&namespace, "namespace", "", "Namespace repo to get status for (multi-repo only, leave unset to get all repos)")
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
		return nil, errors.Wrap(err, "failed to close status file writer with error")
	}

	return ioutil.NopCloser(r), nil
}

// Cmd runs a loop that fetches ACM objects from all available clusters and prints a summary of the
// status of Config Management for each cluster.
var Cmd = &cobra.Command{
	Use: "status",
	// TODO: make Configuration Management a constant (for product renaming)
	Short: `Prints the status of all clusters with Configuration Management installed.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Don't show usage on error, as argument validation passed.
		cmd.SilenceUsage = true

		fmt.Println("Connecting to clusters...")

		clientMap, err := statusClients(flags.Contexts)
		if err != nil {
			// If "no such file or directory" error, unwrap and display before exiting
			if unWrapped := errors.Cause(err); os.IsNotExist(unWrapped) {
				return errors.Wrapf(err, "failed to create client configs")
			}

			glog.Fatalf("Failed to get clients: %v", err)
		}
		if len(clientMap) == 0 {
			return errors.New("no clusters found")
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
		return nil
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
			stateMap[name] = client.clusterStatus(ctx, name, namespace)
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
