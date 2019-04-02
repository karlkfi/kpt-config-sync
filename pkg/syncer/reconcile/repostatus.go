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
	"github.com/google/nomos/pkg/util/repo"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// RepoStatus is a reconciler for maintaining the status field of the Repo resource based upon
// updates from the other syncer reconcilers.
type RepoStatus struct {
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
	// commits is a map of commit token to list of configs currently being reconciled for that commit
	commits map[string][]string
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
func NewRepoStatus(sClient *syncclient.Client) *RepoStatus {
	return &RepoStatus{
		client:  sClient,
		rClient: repo.New(sClient),
		tokens:  make(map[string]bool),
	}
}

// Reconcile is the Reconcile callback for RepoStatus reconciler.
func (r *RepoStatus) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	glog.Infof("Reconcile triggered by %q.", request.NamespacedName)
	ctx := context.Background()
	result := reconcile.Result{}

	repo, sErr := r.rClient.GetOrCreateRepo(ctx)
	if sErr != nil {
		glog.Errorf("Failed to fetch Repo: %v", sErr)
		return result, sErr
	}

	state, err := r.buildState(ctx, repo.Status.Import.Token)
	if err != nil {
		glog.Errorf("Failed to build sync state: %v", err)
		return result, err
	}

	state.merge(&repo.Status, r.latestToken)

	if _, err := r.rClient.UpdateSyncStatus(ctx, repo); err != nil {
		glog.Errorf("Failed to update RepoSyncStatus: %v", err)
		return result, err
	}
	return result, nil
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
		commits: make(map[string][]string),
		configs: make(map[string]configState),
	}

	for _, cc := range ccList.Items {
		state.addConfigToCommit(clusterPrefix(cc.Name), cc.Spec.Token, cc.Status.Token, cc.Status.SyncErrors)
	}
	for _, nc := range ncList.Items {
		state.addConfigToCommit(namespacePrefix(nc.Name), nc.Spec.Token, nc.Status.Token, nc.Status.SyncErrors)
	}

	newTokens := make(map[string]bool)
	for token := range state.commits {
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
		return
	} else {
		// If there are no errors and the tokens do not match, then the importToken indicates the unreconciled commit
		commitHash = importToken
	}
	s.commits[commitHash] = append(s.commits[commitHash], name)
	s.configs[name] = configState{commit: commitHash, errors: errs}
}

// merge updates the given RepoStatus with current configs and commits in the syncState.
func (s syncState) merge(repoStatus *v1.RepoStatus, latestToken string) {
	if repoStatus.Sync.LatestToken != repoStatus.Import.Token && latestToken != "" {
		repoStatus.Sync.LatestToken = latestToken
	}

	var inProgress []v1.RepoSyncChangeStatus
	for token, configNames := range s.commits {
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
