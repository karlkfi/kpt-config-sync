package filesystem

import (
	"context"
	"path/filepath"
	"time"

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
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/util/namespaceconfig"
	"github.com/google/nomos/pkg/util/repo"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const resync = time.Minute * 15
const reconcileTimeout = time.Minute * 5

var _ reconcile.Reconciler = &Reconciler{}

// Reconciler manages Nomos CRs by importing configs from a filesystem tree.
type Reconciler struct {
	configDir             string
	parser                *Parser
	differ                *actions.Differ
	namespaceConfigLister listersv1.NamespaceConfigLister
	clusterConfigLister   listersv1.ClusterConfigLister
	syncLister            listersv1.SyncLister
	repoClient            *repo.Client
	currentDir            string
	lastSyncs             map[string]v1.Sync
}

// NewReconciler returns a new Reconciler.
//
// configDir is the path to the filesystem directory that contains a candidate
// Nomos config directory, which the user intends to be valid but which the
// controller will check for errors.  pollPeriod is the time between two
// successive directory polls. parser is used to convert the contents of
// configDir into a set of Nomos configs.  client is the catch-all client used
// to call configmanagement and other Kubernetes APIs.
func NewReconciler(configDir string, parser *Parser, client meta.Interface, stopChan <-chan struct{}) (*Reconciler, error) {
	if err := configmanagementscheme.AddToScheme(scheme.Scheme); err != nil {
		return nil, errors.Wrapf(err, "filesystem.NewReconciler: can not add to scheme")
	}

	informerFactory := informer.NewSharedInformerFactory(client.ConfigManagement(), resync)
	differ := actions.NewDiffer(
		actions.NewFactories(
			client.ConfigManagement().ConfigmanagementV1(),
			client.ConfigManagement().ConfigmanagementV1(),
			informerFactory.Configmanagement().V1().NamespaceConfigs().Lister(),
			informerFactory.Configmanagement().V1().ClusterConfigs().Lister(),
			informerFactory.Configmanagement().V1().Syncs().Lister()))
	repoClient := repo.NewForImporter(
		client.ConfigManagement().ConfigmanagementV1().Repos(),
		informerFactory.Configmanagement().V1().Repos().Lister())

	// Start informers
	informerFactory.Start(stopChan)
	glog.Infof("Waiting for cache to sync")
	synced := informerFactory.WaitForCacheSync(stopChan)
	for syncType, ok := range synced {
		if !ok {
			elemType := syncType.Elem()
			return nil, errors.Errorf("Failed to sync %s:%s", elemType.PkgPath(), elemType.Name())
		}
	}
	glog.Infof("Caches synced")

	return &Reconciler{
		configDir:             configDir,
		parser:                parser,
		differ:                differ,
		namespaceConfigLister: informerFactory.Configmanagement().V1().NamespaceConfigs().Lister(),
		clusterConfigLister:   informerFactory.Configmanagement().V1().ClusterConfigs().Lister(),
		syncLister:            informerFactory.Configmanagement().V1().Syncs().Lister(),
		repoClient:            repoClient,
	}, nil
}

// Reconcile implements Reconciler.
// It does the following:
//   * Checks for updates to the filesystem that stores config source of truth.
//   * When there are updates, parses the filesystem into AllConfigs, an in-memory
//     representation of desired configs.
//   * Gets the Nomos CRs currently stored in Kubernetes API server.
//   * Compares current and desired Nomos CRs.
//   * Writes updates to make current match desired.
func (c *Reconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	// TODO(b/118385500): Update this check when we support watching/reconciling all Nomos CRs.
	if request.Name != pollFilesystem {
		glog.Errorf("Unexpected reconcile event: %v", request)
		return reconcile.Result{}, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), reconcileTimeout)
	defer cancel()

	glog.V(4).Infof("Reconciling: %v", request)
	startTime := time.Now()

	// Detect whether symlink has changed.
	newDir, err := filepath.EvalSymlinks(c.configDir)
	if err != nil {
		glog.Errorf("Failed to resolve config directory: %v", err)
		importer.Metrics.CycleDuration.WithLabelValues("error").Observe(time.Since(startTime).Seconds())
		sErr := status.SourceError.Errorf("unable to sync repo: %v\n"+
			"Check git-sync logs for more info: kubectl logs -n config-management-system  -l app=git-importer -c git-sync",
			err)
		c.updateSourceStatus(ctx, nil, sErr.ToCME())
		return reconcile.Result{}, nil
	}

	// Check if Syncs have changed on the cluster.  We need to
	// reconcile in case the user removed Syncs that should be there.
	// We don't have a watcher for this event, but rely instead on the
	// poll period to trigger a reconcile.
	currentSyncs, err := namespaceconfig.ListSyncs(c.syncLister)
	if err != nil {
		glog.Errorf("failed quick check of current syncs: %v", err)
		importer.Metrics.CycleDuration.WithLabelValues("error").Observe(time.Since(startTime).Seconds())
		return reconcile.Result{}, nil
	}

	// If last syncs has more sync than current on-cluster sync
	// content, this means that syncs have been removed on the cluster
	// without our intention.
	unchangedSyncs := len(c.differ.SyncsInFirstOnly(c.lastSyncs, currentSyncs)) == 0
	unchangedDir := c.currentDir == newDir

	if unchangedDir && unchangedSyncs {
		glog.V(4).Info("no new changes, nothing to do.")
		return reconcile.Result{}, nil
	}
	glog.Infof("Resolved config dir: %s. Polling config dir: %s", newDir, c.configDir)

	// Parse the commit hash from the new directory to use as an import token.
	token, err := git.CommitHash(newDir)
	if err != nil {
		glog.Warningf("Invalid format for config directory format: %v", err)
		importer.Metrics.CycleDuration.WithLabelValues("error").Observe(time.Since(startTime).Seconds())
		c.updateSourceStatus(ctx, nil, status.SourceError.Errorf("unable to parse commit hash: %v", err).ToCME())
		return reconcile.Result{}, nil
	}

	// Before we start parsing the new directory, update the source token to reflect that this
	// cluster has seen the change even if it runs into issues parsing/importing it.
	repoObj := c.updateSourceStatus(ctx, &token)
	if repoObj == nil {
		glog.Warningf("Repo object is missing. Restarting import of %s.", token)
		// If we failed to get the Repo, restart the controller loop to try to fetch it again.
		return reconcile.Result{}, nil
	}

	currentConfigs, err := namespaceconfig.ListConfigs(c.namespaceConfigLister, c.clusterConfigLister, c.syncLister)
	if err != nil {
		glog.Errorf("failed to list current configs: %v", err)
		importer.Metrics.CycleDuration.WithLabelValues("error").Observe(time.Since(startTime).Seconds())
		return reconcile.Result{}, nil
	}

	// Parse filesystem tree into in-memory NamespaceConfig and ClusterConfig objects.
	desiredConfigs, mErr := c.parser.Parse(newDir, token, currentConfigs, startTime)
	if mErr != nil {
		glog.Warningf("Failed to parse: %v", mErr)
		importer.Metrics.CycleDuration.WithLabelValues("error").Observe(time.Since(startTime).Seconds())
		c.updateImportStatus(ctx, repoObj, token, startTime, status.ToCME(mErr))
		return reconcile.Result{}, nil
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
		c.updateImportStatus(ctx, repoObj, token, startTime, status.ToCME(errs))
		return reconcile.Result{}, nil
	}

	c.currentDir = newDir
	importer.Metrics.CycleDuration.WithLabelValues("success").Observe(time.Since(startTime).Seconds())
	importer.Metrics.NamespaceConfigs.Set(float64(len(desiredConfigs.NamespaceConfigs)))
	c.updateImportStatus(ctx, repoObj, token, startTime, nil)

	c.lastSyncs = desiredConfigs.Syncs
	glog.V(4).Infof("Reconcile completed")
	return reconcile.Result{}, nil
}

// updateImportStatus write an updated RepoImportStatus based upon the given arguments.
func (c *Reconciler) updateImportStatus(ctx context.Context, repoObj *v1.Repo, token string, loadTime time.Time, errs []v1.ConfigManagementError) {
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
func (c *Reconciler) updateSourceStatus(ctx context.Context, token *string, errs ...v1.ConfigManagementError) *v1.Repo {
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
func (c *Reconciler) updateConfigs(current, desired *namespaceconfig.AllConfigs) status.MultiError {
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
