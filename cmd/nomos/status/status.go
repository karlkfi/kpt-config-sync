// Copyright 2022 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/klog/v2"
	"kpt.dev/configsync/cmd/nomos/flags"
	"kpt.dev/configsync/cmd/nomos/util"
	"kpt.dev/configsync/pkg/client/restconfig"
)

const (
	pendingMsg     = "PENDING"
	syncedMsg      = "SYNCED"
	stalledMsg     = "STALLED"
	reconcilingMsg = "RECONCILING"
)

var (
	pollingInterval time.Duration
	namespace       string
	resourceStatus  bool
)

func init() {
	flags.AddContexts(Cmd)
	Cmd.Flags().DurationVar(&flags.ClientTimeout, "timeout", flags.DefaultClusterClientTimeout, "Timeout for connecting to each cluster")
	Cmd.Flags().DurationVar(&pollingInterval, "poll", 0*time.Second, "Polling interval (leave unset to run once)")
	Cmd.Flags().StringVar(&namespace, "namespace", "", "Namespace repo to get status for (multi-repo only, leave unset to get all repos)")
	Cmd.Flags().BoolVar(&resourceStatus, "resources", true, "show resource level status for Namespace repo (multi-repo only)")
}

// GetStatusReadCloser returns a ReadCloser with the output produced by running the "nomos status" command as a string
func GetStatusReadCloser(ctx context.Context, contexts []string) (io.ReadCloser, error) {
	r, w, _ := os.Pipe()
	writer := util.NewWriter(w)

	clientMap, err := ClusterClients(ctx, contexts)
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

		clientMap, err := ClusterClients(cmd.Context(), flags.Contexts)
		if err != nil {
			// If "no such file or directory" error, unwrap and display before exiting
			if unWrapped := errors.Cause(err); os.IsNotExist(unWrapped) {
				return errors.Wrapf(err, "failed to create client configs")
			}

			klog.Fatalf("Failed to get clients: %v", err)
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
func clusterNames(clientMap map[string]*ClusterClient) []string {
	var names []string
	for name := range clientMap {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// clusterStates returns a map of clusterStates calculated from the given map of
// clients, and a list of clusters running in the mono-repo mode.
func clusterStates(ctx context.Context, clientMap map[string]*ClusterClient) (map[string]*ClusterState, []string) {
	stateMap := make(map[string]*ClusterState)
	var monoRepoClusters []string
	for name, client := range clientMap {
		if client == nil {
			stateMap[name] = unavailableCluster(name)
		} else {
			cs := client.clusterStatus(ctx, name, namespace)
			stateMap[name] = cs
			if cs.isMulti != nil && !*cs.isMulti {
				monoRepoClusters = append(monoRepoClusters, name)
			}
		}
	}
	return stateMap, monoRepoClusters
}

// printStatus fetches ConfigManagementStatus and/or RepoStatus from each cluster in the given map
// and then prints a formatted status row for each one. If there are any errors reported by either
// object, those are printed in a second table under the status table.
// nolint:errcheck
func printStatus(ctx context.Context, writer *tabwriter.Writer, clientMap map[string]*ClusterClient, names []string) {
	// First build up a map of all the states to display.
	stateMap, monoRepoClusters := clusterStates(ctx, clientMap)

	// Log a notice for the detected clusters that are running in the mono-repo mode.
	util.MonoRepoNotice(writer, monoRepoClusters...)

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
			state.Ref = "*" + name
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
		klog.Warningf("Failed to execute command: %v", err)
	}
}
