// Package status contains logic for the nomos status CLI command.
package status

import (
	"context"
	"fmt"
	"net"
	"os"
	"sort"
	"strings"
	"sync"

	"github.com/golang/glog"
	"github.com/google/nomos/clientgen/apis"
	typedv1 "github.com/google/nomos/clientgen/apis/typed/configmanagement/v1"
	"github.com/google/nomos/cmd/nomos/util"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	"github.com/google/nomos/pkg/client/restconfig"
	"github.com/google/nomos/pkg/declared"
	"github.com/google/nomos/pkg/reposync"
	"github.com/google/nomos/pkg/rootsync"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

type statusClient struct {
	// TODO(b/170849447): Replace RepoInterface and PodInterface with runtime client usage.
	client           client.Client
	repos            typedv1.RepoInterface
	pods             corev1.PodInterface
	configManagement *util.ConfigManagementClient
}

func (c *statusClient) rootSync(ctx context.Context) (*v1alpha1.RootSync, error) {
	rs := &v1alpha1.RootSync{}
	if err := c.client.Get(ctx, rootsync.ObjectKey(), rs); err != nil {
		return nil, err
	}
	return rs, nil
}

func (c *statusClient) repoSync(ctx context.Context, ns string) (*v1alpha1.RepoSync, error) {
	rs := &v1alpha1.RepoSync{}
	if err := c.client.Get(ctx, reposync.ObjectKey(declared.Scope(ns)), rs); err != nil {
		return nil, err
	}
	return rs, nil
}

func (c *statusClient) repoSyncs(ctx context.Context) ([]*v1alpha1.RepoSync, error) {
	rsl := &v1alpha1.RepoSyncList{}
	if err := c.client.List(ctx, rsl); err != nil {
		return nil, err
	}
	var repoSyncs []*v1alpha1.RepoSync
	for _, rs := range rsl.Items {
		repoSyncs = append(repoSyncs, &rs)
	}
	return repoSyncs, nil
}

// clusterStatus returns the clusterState for the cluster this client is connected to.
func (c *statusClient) clusterStatus(ctx context.Context, cluster, namespace string) *clusterState {
	cs := &clusterState{ref: cluster}

	if !c.isInstalled(cs) {
		return cs
	}
	if !c.isConfigured(cs) {
		return cs
	}

	isMulti, err := c.configManagement.NestedBool("spec", "enableMultiRepo")
	if err != nil {
		cs.status = util.ErrorMsg
		cs.error = err.Error()
		return cs
	}

	if namespace != "" {
		c.namespaceRepoClusterStatus(ctx, cs, namespace)
	} else if isMulti {
		c.multiRepoClusterStatus(ctx, cs)
	} else {
		c.monoRepoClusterStatus(cs)
	}
	return cs
}

// monoRepoClusterStatus populates the given clusterState with the sync status of
// the mono repo on the statusClient's cluster.
func (c *statusClient) monoRepoClusterStatus(cs *clusterState) {
	git, err := c.monoRepoGit()
	if err != nil {
		cs.status = util.ErrorMsg
		cs.error = err.Error()
		return
	}

	repoList, err := c.repos.List(metav1.ListOptions{})
	if err != nil {
		cs.status = util.ErrorMsg
		cs.error = err.Error()
		return
	}

	if len(repoList.Items) == 0 {
		cs.status = util.UnknownMsg
		cs.error = "Repo resource is missing"
		return
	}

	repoStatus := repoList.Items[0].Status
	cs.repos = append(cs.repos, monoRepoStatus(git, repoStatus))
}

// monoRepoGit fetches the mono repo ConfigManagement resource from the cluster
// and builds a Git config out of it.
func (c *statusClient) monoRepoGit() (v1alpha1.Git, error) {
	syncRepo, err := c.configManagement.NestedString("spec", "git", "syncRepo")
	if err != nil {
		return v1alpha1.Git{}, err
	}
	syncBranch, err := c.configManagement.NestedString("spec", "git", "syncBranch")
	if err != nil {
		return v1alpha1.Git{}, err
	}
	syncRev, err := c.configManagement.NestedString("spec", "git", "syncRev")
	if err != nil {
		return v1alpha1.Git{}, err
	}
	policyDir, err := c.configManagement.NestedString("spec", "git", "policyDir")
	if err != nil {
		return v1alpha1.Git{}, err
	}

	return v1alpha1.Git{
		Repo:     syncRepo,
		Branch:   syncBranch,
		Revision: syncRev,
		Dir:      policyDir,
	}, nil
}

// multiRepoClusterStatus populates the given clusterState with the sync status of
// the multi repos on the statusClient's cluster.
func (c *statusClient) multiRepoClusterStatus(ctx context.Context, cs *clusterState) {
	var errs []string
	sync, err := c.rootSync(ctx)
	if err != nil {
		errs = append(errs, err.Error())
	} else {
		cs.repos = append(cs.repos, rootRepoStatus(sync))
	}

	syncs, err := c.repoSyncs(ctx)
	if err != nil {
		errs = append(errs, err.Error())
	} else {
		var repos []*repoState
		for _, rs := range syncs {
			repos = append(repos, namespaceRepoStatus(rs))
		}
		sort.Slice(repos, func(i, j int) bool {
			return repos[i].scope < repos[j].scope
		})
		cs.repos = append(cs.repos, repos...)
	}

	if len(errs) > 0 {
		cs.status = util.ErrorMsg
		cs.error = strings.Join(errs, ", ")
	} else if len(cs.repos) == 0 {
		cs.status = util.UnknownMsg
		cs.error = "No RootSync or RepoSync resources found"
	}
}

// namespaceRepoClusterStatus populates the given clusterState with the sync status of
// the specified namespace repo on the statusClient's cluster.
func (c *statusClient) namespaceRepoClusterStatus(ctx context.Context, cs *clusterState, ns string) {
	sync, err := c.repoSync(ctx, ns)
	if err != nil {
		cs.status = util.ErrorMsg
		cs.error = err.Error()
		return
	}

	cs.repos = append(cs.repos, namespaceRepoStatus(sync))
}

// isInstalled returns true if the statusClient is connected to a cluster where
// Config Sync is installed. Updates the given clusterState with status info if
// Config Sync is not installed.
func (c *statusClient) isInstalled(cs *clusterState) bool {
	podList, err := c.pods.List(metav1.ListOptions{LabelSelector: "k8s-app=config-management-operator"})

	if err != nil {
		if apierrors.IsNotFound(err) {
			cs.status = util.NotInstalledMsg
		} else {
			cs.status = util.ErrorMsg
			cs.error = err.Error()
		}
		return false
	}

	if len(podList.Items) == 0 {
		cs.status = util.NotInstalledMsg
		return false
	}

	return true
}

// isConfigured returns true if the statusClient is connected to a cluster where
// Config Sync is configured. Updates the given clusterState with status info if
// Config Sync is not configured.
func (c *statusClient) isConfigured(cs *clusterState) bool {
	errs, err := c.configManagement.NestedStringSlice("status", "errors")

	if err != nil {
		if apierrors.IsNotFound(err) {
			cs.status = util.NotConfiguredMsg
			cs.error = "ConfigManagement resource is missing"
		} else {
			cs.status = util.ErrorMsg
			cs.error = err.Error()
		}
		return false
	}

	if len(errs) > 0 {
		cs.status = util.NotConfiguredMsg
		cs.error = strings.Join(errs, ", ")
		return false
	}

	return true
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

	s := runtime.NewScheme()
	if sErr := v1.AddToScheme(s); sErr != nil {
		return nil, err
	}
	if sErr := v1alpha1.AddToScheme(s); sErr != nil {
		return nil, err
	}

	for name, cfg := range configs {
		mapper, err := apiutil.NewDynamicRESTMapper(cfg)
		if err != nil {
			fmt.Printf("Failed to create mapper for %q: %v\n", name, err)
			continue
		}

		cl, err := client.New(cfg, client.Options{Scheme: s, Mapper: mapper})
		if err != nil {
			fmt.Printf("Failed to generate runtime client for %q: %v\n", name, err)
			continue
		}

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
					cl,
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
