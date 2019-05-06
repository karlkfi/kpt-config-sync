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

// Controller is controller for managing Nomos CRDs by importing configs from a filesystem tree.
type Controller struct {
	configDir             string
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
//
// configDir is the path to the filesystem directory that contains a candidate
// Nomos config directory, which the user intends to be valid but which the
// controller will check for errors.  pollPeriod is the time between two
// successive directory polls. parser is used to convert the contents of
// configDir into a set of Nomos configs.  client is the catch-all client used
// to call configmanagement and other Kubernetes APIs.  stopChan is a channel
// that the controller will close when it announces that it will stop.
func NewController(configDir string, pollPeriod time.Duration, parser *Parser, client meta.Interface, stopChan chan struct{}) (*Controller, error) {
	if err := configmanagementscheme.AddToScheme(scheme.Scheme); err != nil {
		return nil, errors.Wrapf(err, "filesystem.NewController: can not add to scheme")
	}

	informerFactory := informer.NewSharedInformerFactory(
		client.ConfigManagement(), resync)
	differ := actions.NewDiffer(
		actions.NewFactories(
			client.ConfigManagement().ConfigmanagementV1(),
			client.ConfigManagement().ConfigmanagementV1(),
			informerFactory.Configmanagement().V1().NamespaceConfigs().Lister(),
			informerFactory.Configmanagement().V1().ClusterConfigs().Lister(),
			informerFactory.Configmanagement().V1().Syncs().Lister()))
	repoClient := repo.NewForImporter(client.ConfigManagement().ConfigmanagementV1().Repos(), informerFactory.Configmanagement().V1().Repos().Lister())

	return &Controller{
		configDir:             configDir,
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
	}, nil
}

// Run runs the controller and blocks until an error occurs or stopChan is closed.
//
// Each iteration of the loop does the following:
//   * Checks for updates to the filesystem that stores config source of truth.
//   * When there are updates, parses the filesystem into AllConfigs, an in-memory
//     representation of desired configs.
//   * Gets the configs currently stored in Kubernetes API server.
//   * Compares current and desired configs.
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
			startTime := time.Now()

			// Detect whether symlink has changed.
			newDir, err := filepath.EvalSymlinks(c.configDir)
			if err != nil {
				glog.Errorf("failed to resolve config directory: %v", err)
				importer.Metrics.CycleDuration.WithLabelValues("error").Observe(time.Since(startTime).Seconds())
				c.updateSourceStatus(ctx, nil, status.ToCME(status.From(err)))
				continue
			}

			// Check if Syncs have changed on the cluster.  We need to
			// reconcile in case the user removed Syncs that should be there.
			// We don't have a watcher for this event, but rely instead on the
			// poll period to trigger a reconcile.
			currentSyncs, err := namespaceconfig.ListSyncs(c.syncLister)
			if err != nil {
				glog.Errorf("failed quick check of current syncs: %v", err)
				importer.Metrics.CycleDuration.WithLabelValues("error").Observe(time.Since(startTime).Seconds())
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
			glog.Infof("Resolved config dir: %s. Polling config dir: %s", newDir, c.configDir)

			// Parse the commit hash from the new directory to use as an import token.
			token, err := git.CommitHash(newDir)
			if err != nil {
				glog.Warningf("Failed to parse commit hash: %v", err)
				importer.Metrics.CycleDuration.WithLabelValues("error").Observe(time.Since(startTime).Seconds())
				c.updateSourceStatus(ctx, nil, status.ToCME(status.From(err)))
				continue
			}

			// Before we start parsing the new directory, update the source token to reflect that this
			// cluster has seen the change even if it runs into issues parsing/importing it.
			repoObj := c.updateSourceStatus(ctx, &token, nil)
			if repoObj == nil {
				glog.Warningf("Repo object is missing. Restarting import of %s.", token)
				// If we failed to get the Repo, restart the controller loop to try to fetch it again.
				continue
			}

			currentConfigs, err := namespaceconfig.ListConfigs(c.namespaceConfigLister, c.clusterConfigLister, c.syncLister)
			if err != nil {
				glog.Errorf("failed to list current configs: %v", err)
				importer.Metrics.CycleDuration.WithLabelValues("error").Observe(time.Since(startTime).Seconds())
				continue
			}

			// Parse filesystem tree into in-memory NamespaceConfig and ClusterConfig objects.
			desiredConfigs, mErr := c.parser.Parse(newDir, token, currentConfigs, startTime)
			if mErr != nil {
				glog.Warningf("Failed to parse: %v", mErr)
				importer.Metrics.CycleDuration.WithLabelValues("error").Observe(time.Since(startTime).Seconds())
				c.updateImportStatus(ctx, repoObj, token, startTime, status.ToCME(mErr))
				continue
			}

			// Update the SyncState for all NamespaceConfigs and ClusterConfig.
			for n := range desiredConfigs.NamespaceConfigs {
				pn := desiredConfigs.NamespaceConfigs[n]
				pn.Status.SyncState = v1.StateStale
				desiredConfigs.NamespaceConfigs[n] = pn
			}
			desiredConfigs.ClusterConfig.Status.SyncState = v1.StateStale

			if errs := c.updateConfigs(currentConfigs, desiredConfigs); errs != nil {
				glog.Warningf("Failed to apply actions: %v", errs)
				importer.Metrics.CycleDuration.WithLabelValues("error").Observe(time.Since(startTime).Seconds())
				// TODO(b/126598308): Inspect the actual error type and fully populate the CME fields.
				c.updateImportStatus(ctx, repoObj, token, startTime, status.ToCME(errs))
				continue
			}

			currentDir = newDir
			importer.Metrics.CycleDuration.WithLabelValues("success").Observe(time.Since(startTime).Seconds())
			importer.Metrics.NamespaceConfigs.Set(float64(len(desiredConfigs.NamespaceConfigs)))
			c.updateImportStatus(ctx, repoObj, token, startTime, nil)

			lastSyncs = desiredConfigs.Syncs
			glog.V(4).Infof("pollDir: completed")

		case <-c.stopChan:
			glog.Info("Stop polling")
			return
		}
	}
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
func (c *Controller) updateSourceStatus(ctx context.Context, token *string, errs []v1.ConfigManagementError) *v1.Repo {
	r, err := c.repoClient.GetOrCreateRepo(ctx)
	if err != nil {
		glog.Errorf("failed to get fresh Repo: %v", err)
		return nil
	}
	if token != nil {
		r.Status.Source.Token = *token
	}
	r.Status.Source.Errors = errs

	if _, err = c.repoClient.UpdateSourceStatus(ctx, r); err != nil {
		glog.Errorf("failed to update Repo source status: %v", err)
	}
	return r
}

// updateConfigs calculates and applies the actions needed to go from current to desired.
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
// deleted from configs.
//
// If the same resource and Sync are added again in a subsequent commit, the ordering ensures that
// the resource is restored in config before the Syncer starts managing that type.
func (c *Controller) updateConfigs(current, desired *namespaceconfig.AllConfigs) status.MultiError {
	// Calculate the sequence of actions needed to transition from current to desired state.
	a := c.differ.Diff(*current, *desired)
	return applyActions(a)
}

// applyActions attempts to apply the list of actions provided and returns a slice of all
// errors resulting from the application of those actions
func applyActions(actions []action.Interface) status.MultiError {
	var errs []error
	for _, a := range actions {
		if err := a.Execute(); err != nil {
			errs = append(errs, err)
		}
	}
	// if errs is nil, From returns nil
	return status.From(errs...)
}
