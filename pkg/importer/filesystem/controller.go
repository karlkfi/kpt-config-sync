/*
Copyright 2018 The CSP Config Management Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package filesystem

import (
	"context"
	"path/filepath"
	"time"

	"github.com/google/nomos/pkg/status"

	"github.com/golang/glog"
	configmanagementscheme "github.com/google/nomos/clientgen/apis/scheme"
	"github.com/google/nomos/clientgen/informer"
	listersv1 "github.com/google/nomos/clientgen/listers/configmanagement/v1"
	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/client/action"
	"github.com/google/nomos/pkg/client/meta"
	"github.com/google/nomos/pkg/importer"
	"github.com/google/nomos/pkg/importer/actions"
	"github.com/google/nomos/pkg/importer/git"
	"github.com/google/nomos/pkg/util/namespaceconfig"
	"github.com/google/nomos/pkg/util/repo"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes/scheme"
)

const resync = time.Minute * 15

// Controller is controller for managing Nomos CRDs by importing policies from a filesystem tree.
type Controller struct {
	policyDir             string
	pollPeriod            time.Duration
	parser                *Parser
	differ                *actions.Differ
	discoveryClient       discovery.ServerResourcesInterface
	informerFactory       informer.SharedInformerFactory
	namespaceConfigLister listersv1.NamespaceConfigLister
	clusterConfigLister   listersv1.ClusterConfigLister
	syncLister            listersv1.SyncLister
	repoClient            *repo.Client
	stopChan              chan struct{}
	client                meta.Interface
}

// NewController returns a new Controller.
func NewController(policyDir string, pollPeriod time.Duration, parser *Parser, client meta.Interface, stopChan chan struct{}) *Controller {
	configmanagementscheme.AddToScheme(scheme.Scheme)

	informerFactory := informer.NewSharedInformerFactory(
		client.PolicyHierarchy(), resync)
	differ := actions.NewDiffer(
		actions.NewFactories(
			client.PolicyHierarchy().ConfigmanagementV1(),
			client.PolicyHierarchy().ConfigmanagementV1(),
			informerFactory.Configmanagement().V1().NamespaceConfigs().Lister(),
			informerFactory.Configmanagement().V1().ClusterConfigs().Lister(),
			informerFactory.Configmanagement().V1().Syncs().Lister()))
	repoClient := repo.NewForImporter(client.PolicyHierarchy().ConfigmanagementV1().Repos(), informerFactory.Configmanagement().V1().Repos().Lister())

	return &Controller{
		policyDir:             policyDir,
		pollPeriod:            pollPeriod,
		parser:                parser,
		differ:                differ,
		discoveryClient:       client.Kubernetes().Discovery(),
		informerFactory:       informerFactory,
		namespaceConfigLister: informerFactory.Configmanagement().V1().NamespaceConfigs().Lister(),
		clusterConfigLister:   informerFactory.Configmanagement().V1().ClusterConfigs().Lister(),
		syncLister:            informerFactory.Configmanagement().V1().Syncs().Lister(),
		repoClient:            repoClient,
		stopChan:              stopChan,
		client:                client,
	}
}

// Run runs the controller and blocks until an error occurs or stopChan is closed.
//
// Each iteration of the loop does the following:
//   * Checks for updates to the filesystem that stores policy source of truth.
//   * When there are updates, parses the filesystem into AllPolicies, an in-memory
//     representation of desired policies.
//   * Gets the policies currently stored in Kubernetes API server.
//   * Compares current and desired policies.
//   * Writes updates to make current match desired.
func (c *Controller) Run(ctx context.Context) error {
	// Start informers
	c.informerFactory.Start(c.stopChan)
	glog.Infof("Waiting for cache to sync")
	synced := c.informerFactory.WaitForCacheSync(c.stopChan)
	for syncType, ok := range synced {
		if !ok {
			elemType := syncType.Elem()
			return errors.Errorf("Failed to sync %s:%s", elemType.PkgPath(), elemType.Name())
		}
	}
	glog.Infof("Caches synced")

	c.pollDir(ctx)
	return nil
}

func (c *Controller) pollDir(ctx context.Context) {
	currentDir := ""
	ticker := time.NewTicker(c.pollPeriod)
	// lastSyncs are the syncs that were present during the previous poll loop.
	var lastSyncs map[string]v1.Sync

	for {
		select {
		case <-ticker.C:
			glog.V(4).Infof("pollDir: run")
			// Detect whether symlink has changed.
			newDir, err := filepath.EvalSymlinks(c.policyDir)
			if err != nil {
				glog.Errorf("failed to resolve policydir: %v", err)
				importer.Metrics.PolicyStates.WithLabelValues("failed").Inc()
				c.updateSourceStatus(ctx, nil, cmesForMultiError(status.From(err)))
				continue
			}

			// Check if Syncs have changed on the cluster.  We need to
			// reconcile in case the user removed Syncs that should be there.
			// We don't have a watcher for this event, but rely instead on the
			// poll period to trigger a reconcile.
			currentSyncs, err := namespaceconfig.ListSyncs(c.syncLister)
			if err != nil {
				glog.Errorf("failed quick check of current syncs: %v", err)
				importer.Metrics.PolicyStates.WithLabelValues("list_syncs_failed").Inc()
				continue
			}

			// If last syncs has more sync than current on-cluster sync
			// content, this means that syncs have been removed on the cluster
			// without our intention.
			unchangedSyncs := len(c.differ.SyncsInFirstOnly(lastSyncs, currentSyncs)) == 0
			unchangedDir := currentDir == newDir

			if unchangedDir && unchangedSyncs {
				glog.V(4).Info("pollDir: no new changes, nothing to do.")
				continue
			}
			glog.Infof("Resolved policy dir: %s. Polling policy dir: %s", newDir, c.policyDir)

			// Parse the commit hash from the new directory to use as an import token.
			token, err := git.CommitHash(newDir)
			if err != nil {
				glog.Warningf("Failed to parse commit hash: %v", err)
				importer.Metrics.PolicyStates.WithLabelValues("failed").Inc()
				c.updateSourceStatus(ctx, nil, cmesForMultiError(status.From(err)))
				continue
			}

			// Before we start parsing the new directory, update the source token to reflect that this
			// cluster has seen the change even if it runs into issues parsing/importing it.
			repoObj, err := c.repoClient.GetOrCreateRepo(ctx)
			if err != nil {
				glog.Errorf("failed to get Repo: %v", err)
				continue
			}

			repoObj.Status.Source.Token = token
			c.updateSourceStatus(ctx, &token, nil)

			currentPolicies, err := namespaceconfig.ListPolicies(c.namespaceConfigLister, c.clusterConfigLister, c.syncLister)
			if err != nil {
				glog.Errorf("failed to list current policies: %v", err)
				importer.Metrics.PolicyStates.WithLabelValues("failed").Inc()
				continue
			}

			loadTime := time.Now()
			// Parse filesystem tree into in-memory NamespaceConfig and ClusterConfig objects.
			desiredPolicies, mErr := c.parser.Parse(newDir, token, loadTime)
			if mErr != nil {
				glog.Warningf("Failed to parse: %v", mErr)
				importer.Metrics.PolicyStates.WithLabelValues("failed").Inc()
				c.updateImportStatus(ctx, repoObj, token, loadTime, cmesForMultiError(mErr))
				continue
			}

			// Update the SyncState for all policy nodes and cluster policy.
			for n := range desiredPolicies.NamespaceConfigs {
				pn := desiredPolicies.NamespaceConfigs[n]
				pn.Status.SyncState = v1.StateStale
				desiredPolicies.NamespaceConfigs[n] = pn
			}
			desiredPolicies.ClusterConfig.Status.SyncState = v1.StateStale

			if err := c.updatePolicies(currentPolicies, desiredPolicies); err != nil {
				glog.Warningf("Failed to apply actions: %v", err)
				importer.Metrics.PolicyStates.WithLabelValues("failed").Inc()
				// TODO(b/126598308): Inspect the actual error type and fully populate the CME fields.
				cme := v1.ConfigManagementError{ErrorMessage: err.Error()}
				c.updateImportStatus(ctx, repoObj, token, loadTime, []v1.ConfigManagementError{cme})
				continue
			}

			currentDir = newDir
			importer.Metrics.PolicyStates.WithLabelValues("succeeded").Inc()
			importer.Metrics.Nodes.Set(float64(len(desiredPolicies.NamespaceConfigs)))
			c.updateImportStatus(ctx, repoObj, token, loadTime, nil)

			lastSyncs = desiredPolicies.Syncs
			glog.V(4).Infof("pollDir: completed")

		case <-c.stopChan:
			glog.Info("Stop polling")
			return
		}
	}
}

// cmesForMultiError converts a MultiError into one or more ConfigManagmentErrors.
func cmesForMultiError(mErr status.MultiError) []v1.ConfigManagementError {
	var cmes []v1.ConfigManagementError
	for _, err := range mErr.Errors() {
		// TODO(b/126598308): Inspect the actual error types and fully populate the CME fields.
		cmes = append(cmes, v1.ConfigManagementError{ErrorMessage: err.Error()})
	}
	return cmes
}

// updateImportStatus write an updated RepoImportStatus based upon the given arguments.
func (c *Controller) updateImportStatus(ctx context.Context, repoObj *v1.Repo, token string, loadTime time.Time, errs []v1.ConfigManagementError) {
	// Try to get a fresh copy of Repo since it is has high contention with syncer.
	freshRepoObj, err := c.repoClient.GetOrCreateRepo(ctx)
	if err != nil {
		glog.Errorf("failed to get fresh Repo: %v", err)
	} else {
		repoObj = freshRepoObj
	}

	repoObj.Status.Import.Token = token
	repoObj.Status.Import.LastUpdate = metav1.NewTime(loadTime)
	repoObj.Status.Import.Errors = errs

	if _, err = c.repoClient.UpdateImportStatus(ctx, repoObj); err != nil {
		glog.Errorf("failed to update Repo import status: %v", err)
	}
}

// updateSourceStatus writes the updated Repo.Source.Status field.  A new repo
// is loaded every time before updating.  If errs is nil,
// Repo.Source.Status.Errors will be cleared.  if token is nil, it will not be
// updated so as to preserve any prior content.
func (c *Controller) updateSourceStatus(ctx context.Context, token *string, errs []v1.ConfigManagementError) {
	r, err := c.repoClient.GetOrCreateRepo(ctx)
	if err != nil {
		glog.Errorf("failed to get fresh Repo: %v", err)
		return
	}
	if token != nil {
		r.Status.Source.Token = *token
	}
	r.Status.Source.Errors = errs

	if _, err = c.repoClient.UpdateSourceStatus(ctx, r); err != nil {
		glog.Errorf("failed to update Repo source status: %v", err)
	}
}

// updatePolicies calculates and applies the actions needed to go from current to desired.
// The order of actions is as follows:
//   1. Delete Syncs. This includes any Syncs that are deleted outright, as well as any Syncs that
//      are present in both current and desired, but which lose one or more SyncVersions in the
//      transition.
//   2. Apply NamespaceConfig and ClusterConfig updates.
//   3. Apply remaining Sync updates.
//
// This careful ordering matters in the case where both a Sync and a Resource of the same type are
// deleted in the same commit. The desired outcome is that the resource is not deleted, so we delete
// the Sync first. That way, Syncer stops listening to updates for that type before the resource is
// deleted from policies.
//
// If the same resource and Sync are added again in a subsequent commit, the ordering ensures that
// the resource is restored in policy before the Syncer starts managing that type.
func (c *Controller) updatePolicies(current, desired *namespaceconfig.AllPolicies) error {
	// Calculate the sequence of actions needed to transition from current to desired state.
	a := c.differ.Diff(*current, *desired)
	return applyActions(a)
}

func applyActions(actions []action.Interface) error {
	for _, a := range actions {
		if err := a.Execute(); err != nil {
			return err
		}
	}
	return nil
}
