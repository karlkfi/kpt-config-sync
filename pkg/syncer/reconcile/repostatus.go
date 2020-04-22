package reconcile

import (
	"context"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/golang/glog"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	syncclient "github.com/google/nomos/pkg/syncer/client"
	"github.com/google/nomos/pkg/syncer/metrics"
	"github.com/google/nomos/pkg/util/repo"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// RepoStatus is a reconciler for maintaining the status field of the Repo resource based upon
// updates from the other syncer reconcilers.
type repoStatus struct {
	// ctx is a cancelable ambient context for all reconciler operations.
	ctx context.Context
	// client is used to list configs on the cluster when building state for a commit
	client *syncclient.Client
	// client is used to perform CRUD operations on the Repo resource
	rClient *repo.Client

	// now returns the current time.
	now func() metav1.Time
}

// syncState represents the current status of the syncer and all commits that it is reconciling.
type syncState struct {
	// reconciledCommits is a map of commit tokens that have configs that are already reconciled
	reconciledCommits map[string]bool
	// unreconciledCommits is a map of commit token to list of configs currently being reconciled for
	// that commit
	unreconciledCommits map[string][]string
	// configs is a map of config name to the state of the config being reconciled
	configs map[string]configState

	//resourceConditions contains health status for all resources synced in namespace and cluster configs
	resourceConditions []v1.ResourceCondition
}

// configState represents the current status of a ClusterConfig or NamespaceConfig being reconciled.
type configState struct {
	// commit is the version token of the change to which the config is being reconciled
	commit string
	// errors is a list of any errors that occurred which prevented a successful reconcile
	errors []v1.ConfigManagementError
}

// NewRepoStatus returns a reconciler for maintaining the status field of the Repo resource.
func NewRepoStatus(ctx context.Context, sClient *syncclient.Client, now func() metav1.Time) reconcile.Reconciler {
	return &repoStatus{
		ctx:     ctx,
		client:  sClient,
		rClient: repo.New(sClient),
		now:     now,
	}
}

// Reconcile is the Reconcile callback for RepoStatus reconciler.
func (r *repoStatus) Reconcile(_ reconcile.Request) (reconcile.Result, error) {
	start := r.now()
	metrics.ReconcileEventTimes.WithLabelValues("repo").Set(float64(start.Unix()))

	result, err := r.reconcile()
	metrics.ReconcileDuration.WithLabelValues("repo", metrics.StatusLabel(err)).Observe(time.Since(start.Time).Seconds())

	return result, err
}

func (r *repoStatus) reconcile() (reconcile.Result, error) {
	repoObj, sErr := r.rClient.GetOrCreateRepo(r.ctx)
	if sErr != nil {
		glog.Errorf("Failed to fetch Repo: %v", sErr)
		return reconcile.Result{Requeue: true}, sErr
	}

	state, err := r.buildState(r.ctx, repoObj.Status.Import.Token)
	if err != nil {
		glog.Errorf("Failed to build sync state: %v", err)
		return reconcile.Result{Requeue: true}, sErr
	}

	state.merge(&repoObj.Status, r.now)

	// We used to stop reconciliation here if the sync token is the same as
	// import token.  We no longer do that, to ensure that even non-monotonic
	// status updates are reconciled properly.  See b/131250908 why this is
	// relevant.  Instead, we rely on UpdateSyncStatus to skip updates if the
	// new sync status is equal to the old one.
	updatedRepo, err := r.rClient.UpdateSyncStatus(r.ctx, repoObj)
	if err != nil {
		glog.Errorf("Failed to update RepoSyncStatus: %v", err)
		return reconcile.Result{Requeue: true}, sErr
	}

	// If the ImportToken is different in the updated repo, it means that the importer made a change
	// in the middle of this reconcile. In that case we tell the controller to requeue the request so
	// that we can recalculate sync status with up-to-date information.
	requeue := updatedRepo.Status.Import.Token != repoObj.Status.Import.Token
	return reconcile.Result{Requeue: requeue}, sErr
}

// buildState returns a freshly initialized syncState based upon the current configs on the cluster.
func (r *repoStatus) buildState(ctx context.Context, importToken string) (*syncState, error) {
	ccList := &v1.ClusterConfigList{}
	if err := r.client.List(ctx, ccList); err != nil {
		return nil, errors.Wrapf(err, "listing ClusterConfigs")
	}
	ncList := &v1.NamespaceConfigList{}
	if err := r.client.List(ctx, ncList); err != nil {
		return nil, errors.Wrapf(err, "listing NamespaceConfigs")
	}
	return r.processConfigs(ccList, ncList, importToken), nil
}

// processConfigs is broken out to make unit testing easier.
func (r *repoStatus) processConfigs(ccList *v1.ClusterConfigList, ncList *v1.NamespaceConfigList, importToken string) *syncState {
	state := &syncState{
		reconciledCommits:   make(map[string]bool),
		unreconciledCommits: make(map[string][]string),
		configs:             make(map[string]configState),
	}

	for _, cc := range ccList.Items {
		state.addConfigToCommit(clusterPrefix(cc.Name), cc.Spec.Token, cc.Status.Token, cc.Status.SyncErrors)
		state.resourceConditions = append(state.resourceConditions, cc.Status.ResourceConditions...)
	}
	for _, nc := range ncList.Items {
		state.addConfigToCommit(namespacePrefix(nc.Name), nc.Spec.Token, nc.Status.Token, nc.Status.SyncErrors)
		state.resourceConditions = append(state.resourceConditions, nc.Status.ResourceConditions...)
	}

	return state
}

// addConfigToCommit adds the specified config data to the commit for the specified syncToken.
func (s *syncState) addConfigToCommit(name, importToken, syncToken string, errs []v1.ConfigManagementError) {
	var commitHash string
	if len(errs) > 0 {
		// If there are errors, then the syncToken indicates the unreconciled commit.
		commitHash = syncToken
	} else if importToken == syncToken {
		// If the tokens match and there are no errors, then the config is already done being processed.
		if _, ok := s.unreconciledCommits[syncToken]; !ok {
			s.reconciledCommits[syncToken] = true
		}
		return
	} else {
		// If there are no errors and the tokens do not match, then the importToken indicates the unreconciled commit
		commitHash = importToken
	}
	s.unreconciledCommits[commitHash] = append(s.unreconciledCommits[commitHash], name)
	s.configs[name] = configState{commit: commitHash, errors: errs}
	// If we previously marked the commit as reconciled for a different config, remove the entry.
	if _, ok := s.reconciledCommits[commitHash]; ok {
		delete(s.reconciledCommits, commitHash)
	}
}

// merge updates the given RepoStatus with current configs and commits in the syncState.
func (s syncState) merge(repoStatus *v1.RepoStatus, now func() metav1.Time) {
	var updated bool
	if len(s.unreconciledCommits) == 0 {
		if len(repoStatus.Source.Errors) > 0 || len(repoStatus.Import.Errors) > 0 {
			glog.Infof("No unreconciled commits but there are source/import errors. RepoStatus sync token will remain at %q.", repoStatus.Sync.LatestToken)
		} else if repoStatus.Sync.LatestToken != repoStatus.Import.Token {
			glog.Infof("All commits are reconciled, updating RepoStatus sync token to %q.", repoStatus.Import.Token)
			repoStatus.Sync.LatestToken = repoStatus.Import.Token
			updated = true
		}
	} else {
		glog.Infof("RepoStatus import token at %q, but %d commits are unreconciled. RepoStatus sync token will remain at %q.",
			repoStatus.Import.Token, len(s.unreconciledCommits), repoStatus.Sync.LatestToken)
		if glog.V(2) {
			for token, cfgs := range s.unreconciledCommits {
				glog.Infof("Unreconciled configs for commit %q: %v", token, cfgs)
			}
		}
	}

	var inProgress []v1.RepoSyncChangeStatus
	for token, configNames := range s.unreconciledCommits {
		changeStatus := v1.RepoSyncChangeStatus{Token: token}
		for _, name := range configNames {
			config := s.configs[name]
			changeStatus.Errors = append(changeStatus.Errors, config.errors...)
		}
		inProgress = append(inProgress, changeStatus)
	}

	sort.Slice(inProgress, func(i, j int) bool {
		return strings.Compare(inProgress[i].Token, inProgress[j].Token) < 0
	})

	nonEmpty := len(repoStatus.Sync.InProgress) > 0 || len(inProgress) > 0
	if nonEmpty && !reflect.DeepEqual(repoStatus.Sync.InProgress, inProgress) {
		repoStatus.Sync.InProgress = inProgress
		updated = true
	}

	nonEmpty = len(repoStatus.Sync.ResourceConditions) > 0 || len(s.resourceConditions) > 0
	if nonEmpty && !reflect.DeepEqual(repoStatus.Sync.ResourceConditions, s.resourceConditions) {
		repoStatus.Sync.ResourceConditions = s.resourceConditions
		updated = true
	}

	if updated {
		repoStatus.Sync.LastUpdate = now()
	}
}

// clusterPrefix returns the given name prefixed to indicate it is for a ClusterConfig.
func clusterPrefix(name string) string {
	return "cc:" + name
}

// namespacePrefix returns the given name prefixed to indicate it is for a NamespaceConfig.
func namespacePrefix(name string) string {
	return "nc:" + name
}
