package filesystem

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/google/nomos/pkg/importer/git"
	"github.com/google/nomos/pkg/util/clusterconfig"
	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes/scheme"

	"github.com/golang/glog"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer"
	"github.com/google/nomos/pkg/importer/differ"
	"github.com/google/nomos/pkg/status"
	syncerclient "github.com/google/nomos/pkg/syncer/client"
	"github.com/google/nomos/pkg/syncer/decode"
	utildiscovery "github.com/google/nomos/pkg/util/discovery"
	"github.com/google/nomos/pkg/util/namespaceconfig"
	"github.com/google/nomos/pkg/util/repo"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const reconcileTimeout = time.Minute * 5

var _ reconcile.Reconciler = &Reconciler{}

// Reconciler manages Nomos CRs by importing configs from a filesystem tree.
type Reconciler struct {
	clusterName     string
	gitDir          string
	policyDir       string
	parser          ConfigParser
	client          *syncerclient.Client
	discoveryClient discovery.DiscoveryInterface
	decoder         decode.Decoder
	repoClient      *repo.Client
	cache           cache.Cache
	// appliedGitDir is set to the resolved symlink for the repo once apply has
	// succeeded in order to prevent reprocessing.  On error, this is set to empty
	// string so the importer will retry indefinitely to attempt to recover from
	// an error state.
	appliedGitDir string
}

// NewReconciler returns a new Reconciler.
//
// configDir is the path to the filesystem directory that contains a candidate
// Nomos config directory, which the user intends to be valid but which the
// controller will check for errors.  pollPeriod is the time between two
// successive directory polls. parser is used to convert the contents of
// configDir into a set of Nomos configs.  client is the catch-all client used
// to call configmanagement and other Kubernetes APIs.
func NewReconciler(clusterName string, gitDir string, policyDir string, parser ConfigParser, client *syncerclient.Client,
	discoveryClient discovery.DiscoveryInterface, cache cache.Cache,
	decoder decode.Decoder) (*Reconciler, error) {
	repoClient := repo.New(client)

	return &Reconciler{
		clusterName:     clusterName,
		gitDir:          gitDir,
		policyDir:       policyDir,
		parser:          parser,
		client:          client,
		discoveryClient: discoveryClient,
		repoClient:      repoClient,
		cache:           cache,
		decoder:         decoder,
	}, nil
}

func (c *Reconciler) dirError(ctx context.Context, startTime time.Time, err error) (reconcile.Result, error) {
	glog.Errorf("Failed to resolve config directory: %v", err)
	importer.Metrics.CycleDuration.WithLabelValues("error").Observe(time.Since(startTime).Seconds())
	sErr := status.SourceError.Sprintf("unable to sync repo: %v\n"+
		"Check git-sync logs for more info: kubectl logs -n config-management-system  -l app=git-importer -c git-sync",
		err).Build()
	c.updateSourceStatus(ctx, nil, sErr.ToCME())
	return reconcile.Result{}, nil
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
	ctx, cancel := context.WithTimeout(context.Background(), reconcileTimeout)
	defer cancel()

	glog.V(4).Infof("Reconciling: %v", request)
	startTime := time.Now()

	absGitDir, err := filepath.EvalSymlinks(c.gitDir)
	if err != nil {
		return c.dirError(ctx, startTime, err)
	}

	_, err = os.Stat(filepath.Join(absGitDir, c.policyDir))
	if err != nil {
		return c.dirError(ctx, startTime, err)
	}

	// Detect whether symlink has changed, if the reconcile trigger is to periodically poll the filesystem.
	if request.Name == pollFilesystem && c.appliedGitDir == absGitDir {
		glog.V(4).Info("no new changes, nothing to do.")
		return reconcile.Result{}, nil
	}
	glog.Infof("Resolved config dir: %s. Polling config dir: %s", absGitDir, c.gitDir)
	// Unset applied git dir, only set this on complete import success.
	c.appliedGitDir = ""

	// Parse the commit hash from the new directory to use as an import token.
	token, err := git.CommitHash(absGitDir)
	if err != nil {
		glog.Warningf("Invalid format for config directory format: %v", err)
		importer.Metrics.CycleDuration.WithLabelValues("error").Observe(time.Since(startTime).Seconds())
		c.updateSourceStatus(ctx, nil, status.SourceError.Sprintf("unable to parse commit hash: %v", err).Build().ToCME())
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

	currentConfigs, err := namespaceconfig.ListConfigs(ctx, c.cache)
	if err != nil {
		glog.Errorf("failed to list current configs: %v", err)
		importer.Metrics.CycleDuration.WithLabelValues("error").Observe(time.Since(startTime).Seconds())
		return reconcile.Result{}, nil
	}

	decoder := decode.NewGenericResourceDecoder(scheme.Scheme)
	syncedCRDs, crdErr := clusterconfig.GetCRDs(decoder, currentConfigs.ClusterConfig)
	if crdErr != nil {
		// We were unable to parse the CRDs from the current ClusterConfig, so bail out.
		// TODO(b/146139870): Make error message more user-friendly when this happens.
		return reconcile.Result{}, crdErr
	}

	// Parse filesystem tree into in-memory NamespaceConfig and ClusterConfig objects.
	desiredFileObjects, mErr := c.parser.Parse(syncedCRDs, c.clusterName, true)
	desiredConfigs := namespaceconfig.NewAllConfigs(token, metav1.NewTime(startTime), desiredFileObjects)
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

	if err := c.updateDecoderWithAPIResources(currentConfigs.Syncs, desiredConfigs.Syncs); err != nil {
		glog.Warningf("Failed to parse sync resources: %v", err)
		return reconcile.Result{}, nil
	}

	if errs := differ.Update(ctx, c.client, c.decoder, *currentConfigs, *desiredConfigs); errs != nil {
		glog.Warningf("Failed to apply actions: %v", errs.Error())
		importer.Metrics.CycleDuration.WithLabelValues("error").Observe(time.Since(startTime).Seconds())
		c.updateImportStatus(ctx, repoObj, token, startTime, status.ToCME(errs))
		return reconcile.Result{}, nil
	}

	c.appliedGitDir = absGitDir
	importer.Metrics.CycleDuration.WithLabelValues("success").Observe(time.Since(startTime).Seconds())
	importer.Metrics.NamespaceConfigs.Set(float64(len(desiredConfigs.NamespaceConfigs)))
	c.updateImportStatus(ctx, repoObj, token, startTime, nil)

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

// updateDecoderWithAPIResources uses the discovery API and the set of existing
// syncs on cluster to update the set of resource types the Decoder is able to decode.
func (c *Reconciler) updateDecoderWithAPIResources(syncMaps ...map[string]v1.Sync) error {
	resources, discoveryErr := utildiscovery.GetResources(c.discoveryClient)
	if discoveryErr != nil {
		return discoveryErr
	}

	// We need to populate the scheme with the latest resources on cluster in order to decode GenericResources in
	// NamespaceConfigs and ClusterConfigs.
	apiInfo, err := utildiscovery.NewAPIInfo(resources)
	if err != nil {
		return errors.Wrap(err, "failed to parse server resources")
	}

	var syncList []*v1.Sync
	for _, m := range syncMaps {
		for n := range m {
			sync := m[n]
			syncList = append(syncList, &sync)
		}
	}
	gvks := apiInfo.GroupVersionKinds(syncList...)

	// Update the decoder with all sync-enabled resource types on the cluster.
	c.decoder.UpdateScheme(gvks)
	return nil
}
