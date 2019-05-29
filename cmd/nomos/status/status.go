package status

import (
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"sync"
	"text/tabwriter"
	"time"

	"github.com/golang/glog"
	"github.com/google/nomos/clientgen/apis"
	typedv1 "github.com/google/nomos/clientgen/apis/typed/configmanagement/v1"
	"github.com/google/nomos/cmd/nomos/flags"
	"github.com/google/nomos/cmd/nomos/util"
	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/client/restconfig"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
)

const (
	maxNameLength  = 50
	maxTokenLength = 8
	pendingMsg     = "PENDING"
	syncedMsg      = "SYNCED"
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

// Cmd runs a loop that fetches ACM objects from all available clusters and prints a summary of the
// status of Config Management for each cluster.
var Cmd = &cobra.Command{
	Hidden: true,
	Use:    "status",
	// TODO: make Configuration Management a constant (for product renaming)
	Short: `Prints the status of all clusters with Configuration Management installed.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Connecting to clusters...")

		clientMap, err := statusClients(flags.Contexts)
		if err != nil {
			glog.Fatalf("Failed to get clients: %v", err)
		}
		// Use a sorted order of names to avoid shuffling in the output.
		names := clusterNames(clientMap)

		writer := util.NewWriter(os.Stdout)
		if pollingInterval > 0 {
			for {
				printStatus(writer, clientMap, names)
				time.Sleep(pollingInterval)
			}
		} else {
			printStatus(writer, clientMap, names)
		}
	},
}

// statusClients returns a map of of typed clients keyed by the name of the kubeconfig context they
// are initialized from.
func statusClients(contexts []string) (map[string]*statusClient, error) {
	configs, err := restconfig.AllKubectlConfigs(clientTimeout)
	if configs == nil {
		return nil, errors.Wrap(err, "failed to create client configs")
	}
	if err != nil {
		fmt.Println(err)
	}
	configs = filterConfigs(contexts, configs)

	clientMap := make(map[string]*statusClient)
	unreachableClusters := false

	for name, cfg := range configs {
		policyHierarchyClientSet, err := apis.NewForConfig(cfg)
		if err != nil {
			fmt.Printf("Failed to generate Repo client for %q: %v\n", name, err)
			continue
		}

		k8sClientset, err := kubernetes.NewForConfig(cfg)
		if err != nil {
			fmt.Printf("Failed to generate Kubernetes client for %q: %v\n", name, err)
			continue
		}

		cmClient, err := util.NewConfigManagementClient(cfg)
		if err != nil {
			fmt.Printf("Failed to generate ConfigManagement client for %q: %v\n", name, err)
			continue
		}

		if isReachable(policyHierarchyClientSet, name) {
			clientMap[name] = &statusClient{
				policyHierarchyClientSet.ConfigmanagementV1().Repos(),
				k8sClientset.CoreV1().Pods("kube-system"),
				cmClient,
			}
		} else {
			clientMap[name] = nil
			unreachableClusters = true
		}
	}

	if unreachableClusters {
		// We can't stop the underlying libraries from spamming to glog when a cluster is unreachable,
		// so just flush it out and print a blank line to at least make a clean separation.
		glog.Flush()
		fmt.Println()
	}
	return clientMap, nil
}

// filterConfigs returns the intersection of the given slice and map. If contexts is nil then the
// full map is returned unfiltered.
// TODO: dedup this with the function in version/version.go
func filterConfigs(contexts []string, all map[string]*rest.Config) map[string]*rest.Config {
	if contexts == nil {
		return all
	}
	cfgs := make(map[string]*rest.Config)
	for _, name := range contexts {
		if cfg, ok := all[name]; ok {
			cfgs[name] = cfg
		}
	}
	return cfgs
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
func clusterNames(clientMap map[string]*statusClient) []string {
	var names []string
	var unreachableNames []string
	for name, cl := range clientMap {
		if cl == nil {
			unreachableNames = append(unreachableNames, name)
		} else {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	sort.Strings(unreachableNames)
	return append(names, unreachableNames...)
}

// printStatus fetches ConfigManagementStatus and/or RepoStatus from each cluster in the given map
// and then prints a formatted status row for each one. If there are any errors reported by either
// object, those are printed in a second table under the status table.
// nolint:errcheck
func printStatus(writer *tabwriter.Writer, clientMap map[string]*statusClient, names []string) {
	// First build up maps of all the things we want to display.
	writeMap, errorMap := fetchStatus(clientMap)
	// Now we write everything at once. Processing and then printing helps avoid screen strobe.

	if pollingInterval > 0 {
		// Clear previous output and flush it to avoid messing up column widths.
		clearTerminal(writer)
		writer.Flush()
	}

	// Print table header.
	fmt.Fprintln(writer, "Context\tStatus\tLast Synced Token\t")
	fmt.Fprintln(writer, "-------\t------\t-----------------\t")

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

// fetchStatus returns two maps which are both keyed by cluster name. The first is a map of
// printable cluster status rows and the second is a map of printable cluster error rows.
func fetchStatus(clientMap map[string]*statusClient) (map[string]string, map[string][]string) {
	var mapMutex sync.Mutex
	var wg sync.WaitGroup
	writeMap := make(map[string]string)
	errorMap := make(map[string][]string)

	// We fetch the repo objects in parallel to avoid long delays if multiple clusters are unreachable
	// or slow to respond.
	for name, repoClient := range clientMap {
		if repoClient == nil {
			mapMutex.Lock()
			writeMap[name] = naStatusRow(name)
			mapMutex.Unlock()
			continue
		}
		wg.Add(1)

		go func(name string, sClient *statusClient) {
			status, errs := sClient.clusterStatus(name)

			mapMutex.Lock()
			writeMap[name] = status
			if len(errs) > 0 {
				errorMap[name] = errs
			}
			mapMutex.Unlock()

			wg.Done()
		}(name, repoClient)
	}

	wg.Wait()
	return writeMap, errorMap
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

type statusClient struct {
	repos            typedv1.RepoInterface
	pods             corev1.PodInterface
	configManagement *util.ConfigManagementClient
}

func (c *statusClient) clusterStatus(name string) (status string, errs []string) {
	repoList, err := c.repos.List(metav1.ListOptions{})

	if err == nil && len(repoList.Items) > 0 {
		repoStatus := repoList.Items[0].Status
		status = statusRow(name, repoStatus)
		errs = statusErrors(repoStatus)
		return
	}

	podList, err := c.pods.List(metav1.ListOptions{LabelSelector: "k8s-app=config-management-operator"})

	if err != nil {
		if apierrors.IsNotFound(err) {
			status = util.NotInstalledMsg
		} else {
			status = errorRow(name, util.ErrorMsg)
			errs = append(errs, err.Error())
		}
		return
	} else if len(podList.Items) == 0 {
		status = errorRow(name, util.NotInstalledMsg)
		return
	}

	errs, err = c.configManagement.NestedStringSlice("status", "errors")

	if err != nil {
		if apierrors.IsNotFound(err) {
			status = errorRow(name, util.NotConfiguredMsg)
			errs = append(errs, "ConfigManagement resource is missing")
		} else {
			status = errorRow(name, util.ErrorMsg)
			errs = append(errs, err.Error())
		}
	} else if len(errs) > 0 {
		status = errorRow(name, util.NotConfiguredMsg)
	} else {
		status = errorRow(name, util.UnknownMsg)
	}
	return
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
	if len(token) == 0 {
		token = "N/A"
	} else if len(token) > maxTokenLength {
		token = token[:maxTokenLength]
	}
	return fmt.Sprintf("%s\t%s\t%s\t\n", shortName(name), getStatus(status), token)
}

// naStatusRow returns a printable row for a cluster that is N/A.
func naStatusRow(name string) string {
	return fmt.Sprintf("%s\t%s\n", shortName(name), "N/A")
}

// getStatus returns the given RepoStatus formatted as a short summary string.
func getStatus(status v1.RepoStatus) string {
	if hasErrors(status) {
		return util.ErrorMsg
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
