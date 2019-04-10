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

package reconcile

import (
	"context"
	"sort"
	"strings"

	"github.com/golang/glog"
	"github.com/google/nomos/pkg/api/configmanagement/v1"
	syncclient "github.com/google/nomos/pkg/syncer/client"
	"github.com/google/nomos/pkg/syncer/metrics"
	"github.com/google/nomos/pkg/util/repo"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// RepoStatus is a reconciler for maintaining the status field of the Repo resource based upon
// updates from the other syncer reconcilers.
type RepoStatus struct {
	// ctx is a cancelable ambient context for all reconciler operations.
	ctx context.Context
	// client is used to list configs on the cluster when building state for a commit
	client *syncclient.Client
	// client is used to perform CRUD operations on the Repo resource
	rClient *repo.Client
	// tokens is used to track which tokens have been seen and detect new "latest token"s
	tokens map[string]bool
	// latestToken indicates the most recent version token received by the RepoStatus reconciler
	latestToken string
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
}

// configState represents the current status of a ClusterConfig or PolicyConfig being reconciled.
type configState struct {
	// commit is the version token of the change to which the config is being reconciled
	commit string
	// errors is a list of any errors that occurred which prevented a successful reconcile
	errors []v1.ConfigManagementError
}

// NewRepoStatus returns a reconciler for maintaining the status field of the Repo resource.
func NewRepoStatus(ctx context.Context, sClient *syncclient.Client) *RepoStatus {
	return &RepoStatus{
		ctx:     ctx,
		client:  sClient,
		rClient: repo.New(sClient),
		tokens:  make(map[string]bool),
	}
}

// TODO(b/130295620): Enable linting once we use error interfaces instead of structs.
// Reconcile is the Reconcile callback for RepoStatus reconciler.
// nolint
func (r *RepoStatus) Reconcile(_ reconcile.Request) (reconcile.Result, error) {
	metrics.EventTimes.WithLabelValues("repo-reconcile").Set(float64(now().Unix()))
	timer := prometheus.NewTimer(metrics.RepoReconcileDuration.WithLabelValues())
	defer timer.ObserveDuration()

	result, err := r.reconcile()
	if err != nil {
		return result, err
	}
	// Linting is disabled for this function so that we can return explicit nil for error, which
	// avoids golang type weirdness.
	return result, nil
}

func (r *RepoStatus) reconcile() (reconcile.Result, error) {
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

	state.merge(&repoObj.Status, r.latestToken)

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
func (r *RepoStatus) buildState(ctx context.Context, importToken string) (*syncState, error) {
	opts := client.ListOptions{}
	ccList := &v1.ClusterConfigList{}
	if err := r.client.List(ctx, &opts, ccList); err != nil {
		return nil, errors.Wrapf(err, "listing ClusterConfigs")
	}
	ncList := &v1.NamespaceConfigList{}
	if err := r.client.List(ctx, &opts, ncList); err != nil {
		return nil, errors.Wrapf(err, "listing NamespaceConfigs")
	}
	return r.processConfigs(ccList, ncList, importToken), nil
}

// processConfigs is broken out to make unit testing easier.
func (r *RepoStatus) processConfigs(ccList *v1.ClusterConfigList, ncList *v1.NamespaceConfigList, importToken string) *syncState {
	state := &syncState{
		reconciledCommits:   make(map[string]bool),
		unreconciledCommits: make(map[string][]string),
		configs:             make(map[string]configState),
	}

	for _, cc := range ccList.Items {
		state.addConfigToCommit(clusterPrefix(cc.Name), cc.Spec.Token, cc.Status.Token, cc.Status.SyncErrors)
	}
	for _, nc := range ncList.Items {
		state.addConfigToCommit(namespacePrefix(nc.Name), nc.Spec.Token, nc.Status.Token, nc.Status.SyncErrors)
	}

	if state.reconciledCommits[importToken] {
		r.latestToken = importToken
	} else if _, ok := state.unreconciledCommits[importToken]; ok {
		r.latestToken = importToken
	}

	newTokens := make(map[string]bool)
	for token := range state.unreconciledCommits {
		newTokens[token] = true
		// If we haven't seen a token before, our best guess is that it's the latest token. We compare
		// against the importToken because that is guaranteed to be the latest (but we don't know if the
		// syncer has started reconciling it yet unless we actually see a config from it).
		if !r.tokens[token] && r.latestToken != importToken {
			r.latestToken = token
		}
	}
	r.tokens = newTokens

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
		s.reconciledCommits[syncToken] = true
		return
	} else {
		// If there are no errors and the tokens do not match, then the importToken indicates the unreconciled commit
		commitHash = importToken
	}
	s.unreconciledCommits[commitHash] = append(s.unreconciledCommits[commitHash], name)
	s.configs[name] = configState{commit: commitHash, errors: errs}
}

// merge updates the given RepoStatus with current configs and commits in the syncState.
func (s syncState) merge(repoStatus *v1.RepoStatus, latestToken string) {
	if repoStatus.Sync.LatestToken != repoStatus.Import.Token && latestToken != "" {
		repoStatus.Sync.LatestToken = latestToken
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
	repoStatus.Sync.InProgress = inProgress
	repoStatus.Sync.LastUpdate = now()
}

// clusterPrefix returns the given name prefixed to indicate it is for a ClusterConfig.
func clusterPrefix(name string) string {
	return "cc:" + name
}

// namespacePrefix returns the given name prefixed to indicate it is for a NamespaceConfig.
func namespacePrefix(name string) string {
	return "nc:" + name
}
