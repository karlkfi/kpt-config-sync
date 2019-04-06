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
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/nomos/pkg/api/configmanagement/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const commit1 = "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef"
const commit2 = "feedfacefeedfacefeedfacefeedfacefeedface"
const commit3 = "beadfadebeadfadebeadfadebeadfadebeadfade"

var err1 = v1.ConfigManagementError{ErrorMessage: "KNV9999: oops"}
var err2 = v1.ConfigManagementError{ErrorMessage: "KNV9999: fail"}

func fakeClusterConfig(importToken, syncToken string, errs ...v1.ConfigManagementError) v1.ClusterConfig {
	return v1.ClusterConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: v1.ClusterConfigName,
		},
		Spec: v1.ClusterConfigSpec{
			Token: importToken,
		},
		Status: v1.ClusterConfigStatus{
			Token:      syncToken,
			SyncErrors: errs,
		},
	}
}

func fakeNamespaceConfig(name, importToken, syncToken string, errs ...v1.ConfigManagementError) v1.NamespaceConfig {
	return v1.NamespaceConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: v1.NamespaceConfigSpec{
			Token: importToken,
		},
		Status: v1.NamespaceConfigStatus{
			Token:      syncToken,
			SyncErrors: errs,
		},
	}
}

func TestSyncStateBuilding(t *testing.T) {
	testCases := []struct {
		name      string
		clCfgList *v1.ClusterConfigList
		nsCfgList *v1.NamespaceConfigList
		wantState *syncState
	}{
		{
			name: "build commits that are unreconciled",
			clCfgList: &v1.ClusterConfigList{
				Items: []v1.ClusterConfig{
					fakeClusterConfig(commit2, commit1),
				},
			},
			nsCfgList: &v1.NamespaceConfigList{
				Items: []v1.NamespaceConfig{
					fakeNamespaceConfig("shipping-dev", commit2, commit1),
					fakeNamespaceConfig("audit", commit3, commit2),
				},
			},
			wantState: &syncState{
				reconciledCommits: map[string]bool{},
				unreconciledCommits: map[string][]string{
					commit2: {clusterPrefix(v1.ClusterConfigName), namespacePrefix("shipping-dev")},
					commit3: {namespacePrefix("audit")},
				},
				configs: map[string]configState{
					clusterPrefix(v1.ClusterConfigName): {commit: commit2},
					namespacePrefix("shipping-dev"):     {commit: commit2},
					namespacePrefix("audit"):            {commit: commit3},
				},
			},
		},
		{
			name: "build configs that have reconcile errors",
			clCfgList: &v1.ClusterConfigList{
				Items: []v1.ClusterConfig{
					fakeClusterConfig(commit1, commit1, err1),
				},
			},
			nsCfgList: &v1.NamespaceConfigList{
				Items: []v1.NamespaceConfig{
					fakeNamespaceConfig("shipping-dev", commit2, commit2, err2),
					fakeNamespaceConfig("audit", commit3, commit3, err1),
				},
			},
			wantState: &syncState{
				reconciledCommits: map[string]bool{},
				unreconciledCommits: map[string][]string{
					commit1: {clusterPrefix(v1.ClusterConfigName)},
					commit2: {namespacePrefix("shipping-dev")},
					commit3: {namespacePrefix("audit")},
				},
				configs: map[string]configState{
					clusterPrefix(v1.ClusterConfigName): {commit: commit1, errors: []v1.ConfigManagementError{err1}},
					namespacePrefix("shipping-dev"):     {commit: commit2, errors: []v1.ConfigManagementError{err2}},
					namespacePrefix("audit"):            {commit: commit3, errors: []v1.ConfigManagementError{err1}},
				},
			},
		},
		{
			name: "ignore configs that are already reconciled",
			clCfgList: &v1.ClusterConfigList{
				Items: []v1.ClusterConfig{
					fakeClusterConfig(commit1, commit1),
				},
			},
			nsCfgList: &v1.NamespaceConfigList{
				Items: []v1.NamespaceConfig{
					fakeNamespaceConfig("shipping-dev", commit2, commit2),
					fakeNamespaceConfig("audit", commit3, commit2),
				},
			},
			wantState: &syncState{
				reconciledCommits: map[string]bool{
					commit1: true,
					commit2: true,
				},
				unreconciledCommits: map[string][]string{
					commit3: {namespacePrefix("audit")},
				},
				configs: map[string]configState{
					namespacePrefix("audit"): {commit: commit3},
				},
			},
		},
	}

	cmpOpts := []cmp.Option{
		cmp.AllowUnexported(syncState{}),
		cmp.AllowUnexported(configState{}),
	}
	repoStatus := &RepoStatus{
		tokens: make(map[string]bool),
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			state := repoStatus.processConfigs(tc.clCfgList, tc.nsCfgList, "")

			if diff := cmp.Diff(tc.wantState, state, cmpOpts...); diff != "" {
				t.Errorf("syncState does not match expectation:\n%v", diff)
			}
		})
	}
}

func TestSyncStateMerging(t *testing.T) {
	currentTime := metav1.Now()
	updatedTime := metav1.Time{Time: time.Unix(123, 456)}
	now := func() metav1.Time {
		return updatedTime
	}

	testCases := []struct {
		name   string
		state  *syncState
		status *v1.RepoStatus
		want   *v1.RepoStatus
	}{
		{
			name: "merge state into RepoStatus",
			state: &syncState{
				unreconciledCommits: map[string][]string{
					commit1: {namespacePrefix("shipping-dev")},
					commit2: {namespacePrefix("audit")},
				},
				configs: map[string]configState{
					namespacePrefix("shipping-dev"): {commit: commit1},
					namespacePrefix("audit"):        {commit: commit2, errors: []v1.ConfigManagementError{err1}},
				},
			},
			status: &v1.RepoStatus{
				Source: v1.RepoSourceStatus{
					Token: commit2,
				},
				Import: v1.RepoImportStatus{
					Token:      commit2,
					LastUpdate: currentTime,
				},
				Sync: v1.RepoSyncStatus{
					LatestToken: commit1,
					LastUpdate:  currentTime,
					InProgress: []v1.RepoSyncChangeStatus{
						{Token: commit1},
					},
				},
			},
			want: &v1.RepoStatus{
				Source: v1.RepoSourceStatus{
					Token: commit2,
				},
				Import: v1.RepoImportStatus{
					Token:      commit2,
					LastUpdate: currentTime,
				},
				Sync: v1.RepoSyncStatus{
					LatestToken: commit2,
					LastUpdate:  updatedTime,
					InProgress: []v1.RepoSyncChangeStatus{
						{Token: commit1},
						{Token: commit2, Errors: []v1.ConfigManagementError{err1}},
					},
				},
			},
		},
		{
			name: "merge state and ignore out-of-date version token",
			state: &syncState{
				unreconciledCommits: map[string][]string{
					commit1: {namespacePrefix("shipping-dev")},
				},
				configs: map[string]configState{
					namespacePrefix("shipping-dev"): {commit: commit1, errors: []v1.ConfigManagementError{err1}},
				},
			},
			status: &v1.RepoStatus{
				Source: v1.RepoSourceStatus{
					Token: commit3,
				},
				Import: v1.RepoImportStatus{
					Token:      commit3,
					LastUpdate: currentTime,
				},
				Sync: v1.RepoSyncStatus{
					LatestToken: commit3,
					LastUpdate:  currentTime,
					InProgress: []v1.RepoSyncChangeStatus{
						{Token: commit1},
						{Token: commit2},
					},
				},
			},
			want: &v1.RepoStatus{
				Source: v1.RepoSourceStatus{
					Token: commit3,
				},
				Import: v1.RepoImportStatus{
					Token:      commit3,
					LastUpdate: currentTime,
				},
				Sync: v1.RepoSyncStatus{
					LatestToken: commit3,
					LastUpdate:  updatedTime,
					InProgress: []v1.RepoSyncChangeStatus{
						{Token: commit1, Errors: []v1.ConfigManagementError{err1}},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.state.merge(tc.status, commit2, now)
			if diff := cmp.Diff(tc.want, tc.status); diff != "" {
				t.Errorf("RepoStatus does not match expectation:\n%v", diff)
			}
		})
	}
}
