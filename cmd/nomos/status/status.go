package status

import (
	"fmt"
	"io"
	"io/ioutil"
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
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
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

// GetStatusReadCloser returns a ReadCloser with the output produced by running the "nomos status" command as a string
func GetStatusReadCloser(contexts []string) (io.ReadCloser, error) {
	r, w, _ := os.Pipe()
	writer := util.NewWriter(w)

	clientMap, err := statusClients(contexts)
	if err != nil {
		return nil, err
	}
	names := clusterNames(clientMap)

	printStatus(writer, clientMap, names)
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
//
// TODO(b/131767793) This function (and its children) make up the body of this file, which is far
// too long and lacks unit testing.  To begin, some logic (especially error handling) should be
// extracted from the two commands, placed in pkg/, and unit tested.
func statusClients(contexts []string) (map[string]*statusClient, error) {
	configs, err := restconfig.AllKubectlConfigs(clientTimeout)
	if configs == nil {
		return nil, errors.Wrap(err, "failed to create client configs")
	}
	if err != nil {
		fmt.Println(err)
	}
	configs = filterConfigs(contexts, configs)

	var mapMutex sync.Mutex
	var wg sync.WaitGroup
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

		wg.Add(1)

		go func(pcs *apis.Clientset, kcs *kubernetes.Clientset, cmc *util.ConfigManagementClient, cfgName string) {
			// We need to explicitly check if this code is currently executing
			// on-cluster since the reachability check fails in that case.
			if isOnCluster() || isReachable(pcs, cfgName) {
				mapMutex.Lock()
				clientMap[cfgName] = &statusClient{
					pcs.ConfigmanagementV1().Repos(),
					kcs.CoreV1().Pods(metav1.NamespaceSystem),
					cmc,
				}
				mapMutex.Unlock()
			} else {
				mapMutex.Lock()
				clientMap[cfgName] = nil
				unreachableClusters = true
				mapMutex.Unlock()
			}
			wg.Done()
		}(policyHierarchyClientSet, k8sClientset, cmClient, name)
	}

	wg.Wait()

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

// isOnCluster returns true if the nomos status command is currently being
// executed on a kubernetes cluster. The strategy is based upon
// https://kubernetes.io/docs/concepts/services-networking/connect-applications-service/#environment-variables
func isOnCluster() bool {
	_, onCluster := os.LookupEnv("KUBERNETES_SERVICE_HOST")
	return onCluster
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

	// Prepend an asterisk for the "Current" column, which denotes the users' current context
	currentContext, err := restconfig.CurrentContextName()
	if err != nil {
		fmt.Printf("Failed to get current context name with err: %v\n", errors.Cause(err))
	}
	for _, name := range names {
		if name == currentContext {
			writeMap[name] = "*\t" + writeMap[name]
		} else {
			writeMap[name] = "\t" + writeMap[name]
		}
	}
	// Now we write everything at once. Processing and then printing helps avoid screen strobe.

	if pollingInterval > 0 {
		// Clear previous output and flush it to avoid messing up column widths.
		clearTerminal(writer)
		writer.Flush()
	}

	// Print table header.
	// To prevent column width flickering when Nomos status is in poll mode, we artificially pad
	// the sync status column to the length of the longest status (14 characters)
	fmt.Fprintln(writer, "Current\tContext\tSync Status   \tLast Synced Token\tSync Branch\tResource Status")
	fmt.Fprintln(writer, "-------\t-------\t-----------   \t-----------------\t-----------\t---------------")

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
	syncBranch, err := c.configManagement.NestedString("spec", "git", "syncBranch")
	// TODO(b/131767793): Error logic (and more) needs to be refactored out of status.go
	// and into the pkg directory, where:
	//   a) common code between status.go and version.go can be shared
	//   b) error handling of these two functions can be unified
	//   c) more code can be covered by unit tests
	if err != nil {
		fmt.Printf("Failed to retrieve syncBranch for %q: %v\n", name, err)
	}

	repoList, err := c.repos.List(metav1.ListOptions{})
	if err != nil {
		fmt.Printf("Failed to retrieve repos for %q: %v\n", name, err)
	}

	if err == nil && len(repoList.Items) > 0 {
		repoStatus := repoList.Items[0].Status
		status = statusRow(name, repoStatus, syncBranch)
		errs = syncStatusErrors(repoStatus)
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

// shortName returns a cluster name which has been truncated to the maximum name length.
func shortName(name string) string {
	if len(name) <= maxNameLength {
		return name
	}
	return name[len(name)-maxNameLength:]
}

// errorRow returns the given error message formated as a printable row.
func errorRow(name string, err string) string {
	return stringForRow(name, err, "", "", "")
}

// statusRow returns the given RepoStatus and syncBranch formated as a printable row.
// This consists of (in order) the: Context, Status, Last Synced Token, Sync Branch, Resource Status
func statusRow(name string, status v1.RepoStatus, syncBranch string) string {
	token := status.Sync.LatestToken
	if len(token) == 0 {
		token = "N/A"
	} else if len(token) > maxTokenLength {
		token = token[:maxTokenLength]
	}
	return stringForRow(name, getSyncStatus(status), token, syncBranch, getResourceStatus(status.Sync.ResourceConditions))
}

// naStatusRow returns a printable row for a cluster that is N/A.
func naStatusRow(name string) string {
	return stringForRow(name, "N/A", "", "", "")
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

// stringForRow returns a string containing all of the columns in a particular row, except for
// the 'Current' column which is prepended prior to printing.
// Note: It is necessary to include tabs even for columns which are empty, otherwise the formatting
// of rows below will be misaligned.
func stringForRow(name string, syncStatus string, token string, syncBranch string, resourceStatus v1.ResourceConditionState) string {
	return fmt.Sprintf("%s\t%s\t%s\t%s\t%s\t\n", shortName(name), syncStatus, token, syncBranch, resourceStatus)
}
