package filesystem

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/git"
	"github.com/google/nomos/pkg/util/clusterconfig"
	"github.com/pkg/errors"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
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

var _ reconcile.Reconciler = &reconciler{}

// Reconciler manages Nomos CRs by importing configs from a filesystem tree.
type reconciler struct {
	clusterName     string
	gitDir          string
	policyDir       string
	parser          ConfigParser
	client          *syncerclient.Client
	discoveryClient discovery.DiscoveryInterface
	decoder         decode.Decoder
	repoClient      *repo.Client
	cache           cache.Cache
	// parsedGit is populated when the mounted git repo has successfully been
	// parsed.
	parsedGit *gitState
	// appliedGitDir is set to the resolved symlink for the repo once apply has
	// succeeded in order to prevent reprocessing.  On error, this is set to empty
	// string so the importer will retry indefinitely to attempt to recover from
	// an error state.
	appliedGitDir string
}

// gitState contains the parsed state of the mounted git repo at a certain revision.
type gitState struct {
	// rev is the git revision hash when the git repo was parsed.
	rev string
	// filePathList is a list of the paths of all files parsed from the git repo.
	filePathList []string
	// filePaths is a unified string of the paths in filePathList.
	filePaths string
}

// makeGitState generates a new gitState for the given FileObjects read at the specified revision.
func makeGitState(rev string, objs []ast.FileObject) *gitState {
	gs := &gitState{
		rev:          rev,
		filePathList: make([]string, len(objs)),
	}
	for i, obj := range objs {
		gs.filePathList[i] = obj.SlashPath()
	}
	gs.filePaths = strings.Join(gs.filePathList, ",")
	return gs
}

// dumpForFiles returns a string dump of the given list of FileObjects.
func dumpForFiles(objs []ast.FileObject) string {
	b := strings.Builder{}
	for _, obj := range objs {
		b.WriteString(fmt.Sprintf("%s\n", obj.SlashPath()))
		b.WriteString(fmt.Sprintf("%s %s/%s\n", obj.GroupVersionKind(), obj.GetNamespace(), obj.GetName()))
		b.WriteString(fmt.Sprintf("%v\n", obj.Object))
		b.WriteString("----------\n")
	}
	return b.String()
}

// NewReconciler returns a new Reconciler.
//
// configDir is the path to the filesystem directory that contains a candidate
// Nomos config directory, which the user intends to be valid but which the
// controller will check for errors.  pollPeriod is the time between two
// successive directory polls. parser is used to convert the contents of
// configDir into a set of Nomos configs.  client is the catch-all client used
// to call configmanagement and other Kubernetes APIs.
func newReconciler(clusterName string, gitDir string, policyDir string, parser ConfigParser, client *syncerclient.Client,
	discoveryClient discovery.DiscoveryInterface, cache cache.Cache,
	decoder decode.Decoder) (*reconciler, error) {
	repoClient := repo.New(client)

	return &reconciler{
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

// dirError updates repo source status with an error due to failure to read mounted git repo.
func (c *reconciler) dirError(ctx context.Context, startTime time.Time, err error) (reconcile.Result, error) {
	glog.Errorf("Failed to resolve config directory: %v", err)
	importer.Metrics.CycleDuration.WithLabelValues("error").Observe(time.Since(startTime).Seconds())
	sErr := status.SourceError.Sprintf("unable to sync repo: %v\n"+
		"Check git-sync logs for more info: kubectl logs -n config-management-system  -l app=git-importer -c git-sync",
		err).Build()
	c.updateSourceStatus(ctx, nil, sErr.ToCME())
	return reconcile.Result{}, nil
}

// filesystemError updates repo source status with an error due to inconsistent read from filesystem.
func (c *reconciler) filesystemError(ctx context.Context, rev string) (reconcile.Result, error) {
	sErr := status.SourceError.Sprintf("inconsistent files read from mounted git repo at revision %s", rev).Build()
	c.updateSourceStatus(ctx, &rev, sErr.ToCME())
	return reconcile.Result{}, sErr
}

// Reconcile implements Reconciler.
// It does the following:
//   * Checks for updates to the filesystem that stores config source of truth.
//   * When there are updates, parses the filesystem into AllConfigs, an in-memory
//     representation of desired configs.
//   * Gets the Nomos CRs currently stored in Kubernetes API server.
//   * Compares current and desired Nomos CRs.
//   * Writes updates to make current match desired.
func (c *reconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
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

	getSyncedCRDs := func() ([]*v1beta1.CustomResourceDefinition, status.MultiError) {
		// Don't preemptively get the synced CRDs since we may not need them.
		decoder := decode.NewGenericResourceDecoder(scheme.Scheme)
		return clusterconfig.GetCRDs(decoder, currentConfigs.ClusterConfig)
	}

	// check git status, blow up if we see issues
	if err := git.CheckClean(absGitDir); err != nil {
		glog.Errorf("git check clean returned error: %v", err)
		LogWalkDirectory(absGitDir)
		return reconcile.Result{}, err
	}

	// Parse filesystem tree into in-memory NamespaceConfig and ClusterConfig objects.
	desiredFileObjects, mErr := c.parser.Parse(c.clusterName, true, getSyncedCRDs)
	if mErr != nil {
		glog.Warningf("Failed to parse: %v", mErr)
		importer.Metrics.CycleDuration.WithLabelValues("error").Observe(time.Since(startTime).Seconds())
		c.updateImportStatus(ctx, repoObj, token, startTime, status.ToCME(mErr))
		return reconcile.Result{}, nil
	}

	gs := makeGitState(absGitDir, desiredFileObjects)
	if c.parsedGit == nil {
		glog.Infof("Importer state initialized at git revision %s. Unverified file list: %s", gs.rev, gs.filePaths)
	} else if c.parsedGit.rev != gs.rev {
		glog.Infof("Importer state updated to git revision %s. Unverified files list: %s", gs.rev, gs.filePaths)
	} else if c.parsedGit.filePaths == gs.filePaths {
		glog.V(2).Infof("Importer state remains at git revision %s. Verified files hash: %s", gs.rev, gs.filePaths)
	} else {
		glog.Errorf("Importer read inconsistent files at git revision %s.\nExpected files hash: %s\nDiff: %s", gs.rev, c.parsedGit.filePaths, cmp.Diff(c.parsedGit.filePathList, gs.filePathList))
		glog.Errorf("Inconsistent files:\n%s", dumpForFiles(desiredFileObjects))
		return c.filesystemError(ctx, absGitDir)
	}
	c.parsedGit = gs

	desiredConfigs := namespaceconfig.NewAllConfigs(token, metav1.NewTime(startTime), desiredFileObjects)
	if sErr := c.sanityCheck(desiredConfigs, currentConfigs); sErr != nil {
		c.updateSourceStatus(ctx, &token, sErr.ToCME())
		return reconcile.Result{}, sErr
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

// sanityCheck reports if the importer would cause the cluster to drop to zero
// NamespaceConfigs from anything other than zero or one on the cluster currently.
// That is too dangerous of a change to actually carry out.
func (c *reconciler) sanityCheck(desired, current *namespaceconfig.AllConfigs) status.Error {
	if len(desired.NamespaceConfigs) == 0 {
		count := len(current.NamespaceConfigs)
		if count > 1 {
			glog.Errorf("Importer parsed 0 NamespaceConfigs from mounted git repo but detected %d NamespaceConfigs on the cluster. This is a dangerous change, so it will be rejected.", count)
			return status.EmptySourceError(count)
		}
		glog.Warningf("Importer did not parse any NamespaceConfigs in git repo. Cluster currently has %d NamespaceConfigs, so this will proceed.", count)
	}
	return nil
}

// updateImportStatus write an updated RepoImportStatus based upon the given arguments.
func (c *reconciler) updateImportStatus(ctx context.Context, repoObj *v1.Repo, token string, loadTime time.Time, errs []v1.ConfigManagementError) {
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
func (c *reconciler) updateSourceStatus(ctx context.Context, token *string, errs ...v1.ConfigManagementError) *v1.Repo {
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
func (c *reconciler) updateDecoderWithAPIResources(syncMaps ...map[string]v1.Sync) error {
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
