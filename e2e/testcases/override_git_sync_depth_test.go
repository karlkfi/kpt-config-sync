// Copyright 2022 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package e2e

import (
	"testing"
	"time"

	"github.com/google/nomos/e2e/nomostest"
	"github.com/google/nomos/e2e/nomostest/ntopts"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/api/configsync"
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	"github.com/google/nomos/pkg/api/configsync/v1beta1"
	ocmetrics "github.com/google/nomos/pkg/metrics"
	"github.com/google/nomos/pkg/reconciler"
	"github.com/google/nomos/pkg/reconcilermanager"
	"github.com/google/nomos/pkg/reconcilermanager/controllers"
	"github.com/google/nomos/pkg/testing/fake"
	corev1 "k8s.io/api/core/v1"
)

func TestOverrideGitSyncDepthV1Alpha1(t *testing.T) {
	nt := nomostest.New(t, ntopts.SkipMonoRepo,
		ntopts.NamespaceRepo(backendNamespace, configsync.RepoSyncName),
		ntopts.NamespaceRepo(frontendNamespace, configsync.RepoSyncName))
	nt.WaitForRepoSyncs()

	rootReconcilerGitSyncCM := controllers.ReconcilerResourceName(nomostest.DefaultRootReconcilerName, reconcilermanager.GitSync)
	nsReconcilerBackendGitSyncCM := controllers.ReconcilerResourceName(reconciler.NsReconcilerName(backendNamespace, configsync.RepoSyncName), reconcilermanager.GitSync)
	nsReconcilerFrontendGitSyncCM := controllers.ReconcilerResourceName(reconciler.NsReconcilerName(frontendNamespace, configsync.RepoSyncName), reconcilermanager.GitSync)
	key := "GIT_SYNC_DEPTH"
	defaultDepth := controllers.SyncDepthNoRev

	// Verify the root-reconciler-git-sync ConfigMap has the correct git sync depth setting.
	err := nt.Validate(rootReconcilerGitSyncCM, v1.NSConfigManagementSystem, &corev1.ConfigMap{},
		nomostest.HasKeyValuePairInConfigMapData(key, defaultDepth))
	if err != nil {
		nt.T.Fatal(err)
	}

	// Verify the ns-reconciler-backend-git-sync ConfigMap has the correct git sync depth setting.
	err = nt.Validate(nsReconcilerBackendGitSyncCM, v1.NSConfigManagementSystem, &corev1.ConfigMap{},
		nomostest.HasKeyValuePairInConfigMapData(key, defaultDepth))
	if err != nil {
		nt.T.Fatal(err)
	}

	// Verify the ns-reconciler-frontend-git-sync ConfigMap has the correct git sync depth setting.
	err = nt.Validate(nsReconcilerFrontendGitSyncCM, v1.NSConfigManagementSystem, &corev1.ConfigMap{},
		nomostest.HasKeyValuePairInConfigMapData(key, defaultDepth))
	if err != nil {
		nt.T.Fatal(err)
	}

	err = nt.ValidateMetrics(nomostest.SyncMetricsToLatestCommit(nt), func() error {
		return nt.ValidateMetricNotFound(ocmetrics.GitSyncDepthOverrideCountView.Name)
	})
	if err != nil {
		nt.T.Fatal(err)
	}

	nn := nomostest.RepoSyncNN(backendNamespace, configsync.RepoSyncName)
	repo, exist := nt.NonRootRepos[nn]
	if !exist {
		nt.T.Fatal("nonexistent repo")
	}
	rootSync := fake.RootSyncObjectV1Alpha1(configsync.RootSyncName)
	repoSyncBackend := nomostest.RepoSyncObjectV1Alpha1(nn.Namespace, nn.Name, nt.GitProvider.SyncURL(repo.RemoteRepoName))

	// Override the git sync depth setting for root-reconciler
	nt.MustMergePatch(rootSync, `{"spec": {"override": {"gitSyncDepth": 5}}}`)

	// Verify the root-reconciler-git-sync ConfigMap has the correct git sync depth setting.
	_, err = nomostest.Retry(30*time.Second, func() error {
		return nt.Validate(rootReconcilerGitSyncCM, v1.NSConfigManagementSystem, &corev1.ConfigMap{},
			nomostest.HasKeyValuePairInConfigMapData(key, "5"))
	})
	if err != nil {
		nt.T.Fatal(err)
	}

	// Verify the ns-reconciler-backend-git-sync ConfigMap has the correct git sync depth setting.
	err = nt.Validate(nsReconcilerBackendGitSyncCM, v1.NSConfigManagementSystem, &corev1.ConfigMap{},
		nomostest.HasKeyValuePairInConfigMapData(key, defaultDepth))
	if err != nil {
		nt.T.Fatal(err)
	}

	// Verify the ns-reconciler-frontend-git-sync ConfigMap has the correct git sync depth setting.
	err = nt.Validate(nsReconcilerFrontendGitSyncCM, v1.NSConfigManagementSystem, &corev1.ConfigMap{},
		nomostest.HasKeyValuePairInConfigMapData(key, defaultDepth))
	if err != nil {
		nt.T.Fatal(err)
	}

	// Override the git sync depth setting for ns-reconciler-backend
	var depth int64 = 33
	repoSyncBackend.Spec.Override.GitSyncDepth = &depth
	nt.RootRepos[configsync.RootSyncName].Add(nomostest.StructuredNSPath(backendNamespace, configsync.RepoSyncName), repoSyncBackend)
	nt.RootRepos[configsync.RootSyncName].CommitAndPush("Update backend RepoSync git sync depth to 33")
	nt.WaitForRepoSyncs()

	// Verify the ns-reconciler-backend-git-sync ConfigMap has the correct git sync depth setting.
	_, err = nomostest.Retry(30*time.Second, func() error {
		return nt.Validate(nsReconcilerBackendGitSyncCM, v1.NSConfigManagementSystem, &corev1.ConfigMap{},
			nomostest.HasKeyValuePairInConfigMapData(key, "33"))
	})
	if err != nil {
		nt.T.Fatal(err)
	}

	// Verify the root-reconciler-git-sync ConfigMap has the correct git sync depth setting.
	err = nt.Validate(rootReconcilerGitSyncCM, v1.NSConfigManagementSystem, &corev1.ConfigMap{},
		nomostest.HasKeyValuePairInConfigMapData(key, "5"))
	if err != nil {
		nt.T.Fatal(err)
	}

	// Verify the ns-reconciler-frontend-git-sync ConfigMap has the correct git sync depth setting.
	err = nt.Validate(nsReconcilerFrontendGitSyncCM, v1.NSConfigManagementSystem, &corev1.ConfigMap{},
		nomostest.HasKeyValuePairInConfigMapData(key, defaultDepth))
	if err != nil {
		nt.T.Fatal(err)
	}

	// Override the git sync depth setting for root-reconciler to 0
	nt.MustMergePatch(rootSync, `{"spec": {"override": {"gitSyncDepth": 0}}}`)

	// Verify the root-reconciler-git-sync ConfigMap has the correct git sync depth setting.
	_, err = nomostest.Retry(30*time.Second, func() error {
		return nt.Validate(rootReconcilerGitSyncCM, v1.NSConfigManagementSystem, &corev1.ConfigMap{},
			nomostest.HasKeyValuePairInConfigMapData(key, "0"))
	})
	if err != nil {
		nt.T.Fatal(err)
	}

	err = nt.ValidateMetrics(nomostest.SyncMetricsToLatestCommit(nt), func() error {
		return nt.ValidateGitSyncDepthOverrideCount(2)
	})
	if err != nil {
		nt.T.Fatal(err)
	}

	// Clear `spec.override` from the RootSync
	nt.MustMergePatch(rootSync, `{"spec": {"override": null}}`)

	// Verify the root-reconciler-git-sync ConfigMap has the correct git sync depth setting.
	_, err = nomostest.Retry(30*time.Second, func() error {
		return nt.Validate(rootReconcilerGitSyncCM, v1.NSConfigManagementSystem, &corev1.ConfigMap{},
			nomostest.HasKeyValuePairInConfigMapData(key, defaultDepth))
	})
	if err != nil {
		nt.T.Fatal(err)
	}

	// Verify the ns-reconciler-backend-git-sync ConfigMap has the correct git sync depth setting.
	err = nt.Validate(nsReconcilerBackendGitSyncCM, v1.NSConfigManagementSystem, &corev1.ConfigMap{},
		nomostest.HasKeyValuePairInConfigMapData(key, "33"))
	if err != nil {
		nt.T.Fatal(err)
	}

	// Verify the ns-reconciler-frontend-git-sync ConfigMap has the correct git sync depth setting.
	err = nt.Validate(nsReconcilerFrontendGitSyncCM, v1.NSConfigManagementSystem, &corev1.ConfigMap{},
		nomostest.HasKeyValuePairInConfigMapData(key, defaultDepth))
	if err != nil {
		nt.T.Fatal(err)
	}

	// Override the git sync depth setting for ns-reconciler-backend to 0
	depth = 0
	repoSyncBackend.Spec.Override.GitSyncDepth = &depth
	nt.RootRepos[configsync.RootSyncName].Add(nomostest.StructuredNSPath(backendNamespace, configsync.RepoSyncName), repoSyncBackend)
	nt.RootRepos[configsync.RootSyncName].CommitAndPush("Update backend RepoSync git sync depth to 0")
	nt.WaitForRepoSyncs()

	// Verify the ns-reconciler-backend-git-sync ConfigMap has the correct git sync depth setting.
	_, err = nomostest.Retry(30*time.Second, func() error {
		return nt.Validate(nsReconcilerBackendGitSyncCM, v1.NSConfigManagementSystem, &corev1.ConfigMap{},
			nomostest.HasKeyValuePairInConfigMapData(key, "0"))
	})
	if err != nil {
		nt.T.Fatal(err)
	}

	// Clear `spec.override` from repoSyncBackend
	repoSyncBackend.Spec.Override = v1alpha1.OverrideSpec{}
	nt.RootRepos[configsync.RootSyncName].Add(nomostest.StructuredNSPath(backendNamespace, configsync.RepoSyncName), repoSyncBackend)
	nt.RootRepos[configsync.RootSyncName].CommitAndPush("Clear `spec.override` from repoSyncBackend")
	nt.WaitForRepoSyncs()

	// Verify the ns-reconciler-backend-git-sync ConfigMap has the correct git sync depth setting.
	_, err = nomostest.Retry(30*time.Second, func() error {
		return nt.Validate(nsReconcilerBackendGitSyncCM, v1.NSConfigManagementSystem, &corev1.ConfigMap{},
			nomostest.HasKeyValuePairInConfigMapData(key, defaultDepth))
	})
	if err != nil {
		nt.T.Fatal(err)
	}

	// Verify the root-reconciler-git-sync ConfigMap has the correct git sync depth setting.
	err = nt.Validate(rootReconcilerGitSyncCM, v1.NSConfigManagementSystem, &corev1.ConfigMap{},
		nomostest.HasKeyValuePairInConfigMapData(key, defaultDepth))
	if err != nil {
		nt.T.Fatal(err)
	}

	// Verify the ns-reconciler-frontend-git-sync ConfigMap has the correct git sync depth setting.
	err = nt.Validate(nsReconcilerFrontendGitSyncCM, v1.NSConfigManagementSystem, &corev1.ConfigMap{},
		nomostest.HasKeyValuePairInConfigMapData(key, defaultDepth))
	if err != nil {
		nt.T.Fatal(err)
	}

	err = nt.ValidateMetrics(nomostest.SyncMetricsToLatestCommit(nt), func() error {
		return nt.ValidateMetricNotFound(ocmetrics.GitSyncDepthOverrideCountView.Name)
	})
	if err != nil {
		nt.T.Fatal(err)
	}
}

func TestOverrideGitSyncDepthV1Beta1(t *testing.T) {
	nt := nomostest.New(t, ntopts.SkipMonoRepo,
		ntopts.NamespaceRepo(backendNamespace, configsync.RepoSyncName),
		ntopts.NamespaceRepo(frontendNamespace, configsync.RepoSyncName))
	nt.WaitForRepoSyncs()

	rootReconcilerGitSyncCM := controllers.ReconcilerResourceName(nomostest.DefaultRootReconcilerName, reconcilermanager.GitSync)
	nsReconcilerBackendGitSyncCM := controllers.ReconcilerResourceName(reconciler.NsReconcilerName(backendNamespace, configsync.RepoSyncName), reconcilermanager.GitSync)
	nsReconcilerFrontendGitSyncCM := controllers.ReconcilerResourceName(reconciler.NsReconcilerName(frontendNamespace, configsync.RepoSyncName), reconcilermanager.GitSync)
	key := "GIT_SYNC_DEPTH"
	defaultDepth := controllers.SyncDepthNoRev

	// Verify the root-reconciler-git-sync ConfigMap has the correct git sync depth setting.
	err := nt.Validate(rootReconcilerGitSyncCM, v1.NSConfigManagementSystem, &corev1.ConfigMap{},
		nomostest.HasKeyValuePairInConfigMapData(key, defaultDepth))
	if err != nil {
		nt.T.Fatal(err)
	}

	// Verify the ns-reconciler-backend-git-sync ConfigMap has the correct git sync depth setting.
	err = nt.Validate(nsReconcilerBackendGitSyncCM, v1.NSConfigManagementSystem, &corev1.ConfigMap{},
		nomostest.HasKeyValuePairInConfigMapData(key, defaultDepth))
	if err != nil {
		nt.T.Fatal(err)
	}

	// Verify the ns-reconciler-frontend-git-sync ConfigMap has the correct git sync depth setting.
	err = nt.Validate(nsReconcilerFrontendGitSyncCM, v1.NSConfigManagementSystem, &corev1.ConfigMap{},
		nomostest.HasKeyValuePairInConfigMapData(key, defaultDepth))
	if err != nil {
		nt.T.Fatal(err)
	}

	err = nt.ValidateMetrics(nomostest.SyncMetricsToLatestCommit(nt), func() error {
		return nt.ValidateMetricNotFound(ocmetrics.GitSyncDepthOverrideCountView.Name)
	})
	if err != nil {
		nt.T.Fatal(err)
	}

	nn := nomostest.RepoSyncNN(backendNamespace, configsync.RepoSyncName)
	repo, exist := nt.NonRootRepos[nn]
	if !exist {
		nt.T.Fatal("nonexistent repo")
	}
	rootSync := fake.RootSyncObjectV1Beta1(configsync.RootSyncName)
	repoSyncBackend := nomostest.RepoSyncObjectV1Beta1(nn.Namespace, nn.Name, nt.GitProvider.SyncURL(repo.RemoteRepoName))

	// Override the git sync depth setting for root-reconciler
	nt.MustMergePatch(rootSync, `{"spec": {"override": {"gitSyncDepth": 5}}}`)

	// Verify the root-reconciler-git-sync ConfigMap has the correct git sync depth setting.
	_, err = nomostest.Retry(30*time.Second, func() error {
		return nt.Validate(rootReconcilerGitSyncCM, v1.NSConfigManagementSystem, &corev1.ConfigMap{},
			nomostest.HasKeyValuePairInConfigMapData(key, "5"))
	})
	if err != nil {
		nt.T.Fatal(err)
	}

	// Verify the ns-reconciler-backend-git-sync ConfigMap has the correct git sync depth setting.
	err = nt.Validate(nsReconcilerBackendGitSyncCM, v1.NSConfigManagementSystem, &corev1.ConfigMap{},
		nomostest.HasKeyValuePairInConfigMapData(key, defaultDepth))
	if err != nil {
		nt.T.Fatal(err)
	}

	// Verify the ns-reconciler-frontend-git-sync ConfigMap has the correct git sync depth setting.
	err = nt.Validate(nsReconcilerFrontendGitSyncCM, v1.NSConfigManagementSystem, &corev1.ConfigMap{},
		nomostest.HasKeyValuePairInConfigMapData(key, defaultDepth))
	if err != nil {
		nt.T.Fatal(err)
	}

	// Override the git sync depth setting for ns-reconciler-backend
	var depth int64 = 33
	repoSyncBackend.Spec.Override.GitSyncDepth = &depth
	nt.RootRepos[configsync.RootSyncName].Add(nomostest.StructuredNSPath(backendNamespace, configsync.RepoSyncName), repoSyncBackend)
	nt.RootRepos[configsync.RootSyncName].CommitAndPush("Update backend RepoSync git sync depth to 33")
	nt.WaitForRepoSyncs()

	// Verify the ns-reconciler-backend-git-sync ConfigMap has the correct git sync depth setting.
	_, err = nomostest.Retry(30*time.Second, func() error {
		return nt.Validate(nsReconcilerBackendGitSyncCM, v1.NSConfigManagementSystem, &corev1.ConfigMap{},
			nomostest.HasKeyValuePairInConfigMapData(key, "33"))
	})
	if err != nil {
		nt.T.Fatal(err)
	}

	// Verify the root-reconciler-git-sync ConfigMap has the correct git sync depth setting.
	err = nt.Validate(rootReconcilerGitSyncCM, v1.NSConfigManagementSystem, &corev1.ConfigMap{},
		nomostest.HasKeyValuePairInConfigMapData(key, "5"))
	if err != nil {
		nt.T.Fatal(err)
	}

	// Verify the ns-reconciler-frontend-git-sync ConfigMap has the correct git sync depth setting.
	err = nt.Validate(nsReconcilerFrontendGitSyncCM, v1.NSConfigManagementSystem, &corev1.ConfigMap{},
		nomostest.HasKeyValuePairInConfigMapData(key, defaultDepth))
	if err != nil {
		nt.T.Fatal(err)
	}

	// Override the git sync depth setting for root-reconciler to 0
	nt.MustMergePatch(rootSync, `{"spec": {"override": {"gitSyncDepth": 0}}}`)

	// Verify the root-reconciler-git-sync ConfigMap has the correct git sync depth setting.
	_, err = nomostest.Retry(30*time.Second, func() error {
		return nt.Validate(rootReconcilerGitSyncCM, v1.NSConfigManagementSystem, &corev1.ConfigMap{},
			nomostest.HasKeyValuePairInConfigMapData(key, "0"))
	})
	if err != nil {
		nt.T.Fatal(err)
	}

	err = nt.ValidateMetrics(nomostest.SyncMetricsToLatestCommit(nt), func() error {
		return nt.ValidateGitSyncDepthOverrideCount(2)
	})
	if err != nil {
		nt.T.Fatal(err)
	}

	// Clear `spec.override` from the RootSync
	nt.MustMergePatch(rootSync, `{"spec": {"override": null}}`)

	// Verify the root-reconciler-git-sync ConfigMap has the correct git sync depth setting.
	_, err = nomostest.Retry(30*time.Second, func() error {
		return nt.Validate(rootReconcilerGitSyncCM, v1.NSConfigManagementSystem, &corev1.ConfigMap{},
			nomostest.HasKeyValuePairInConfigMapData(key, defaultDepth))
	})
	if err != nil {
		nt.T.Fatal(err)
	}

	// Verify the ns-reconciler-backend-git-sync ConfigMap has the correct git sync depth setting.
	err = nt.Validate(nsReconcilerBackendGitSyncCM, v1.NSConfigManagementSystem, &corev1.ConfigMap{},
		nomostest.HasKeyValuePairInConfigMapData(key, "33"))
	if err != nil {
		nt.T.Fatal(err)
	}

	// Verify the ns-reconciler-frontend-git-sync ConfigMap has the correct git sync depth setting.
	err = nt.Validate(nsReconcilerFrontendGitSyncCM, v1.NSConfigManagementSystem, &corev1.ConfigMap{},
		nomostest.HasKeyValuePairInConfigMapData(key, defaultDepth))
	if err != nil {
		nt.T.Fatal(err)
	}

	// Override the git sync depth setting for ns-reconciler-backend to 0
	depth = 0
	repoSyncBackend.Spec.Override.GitSyncDepth = &depth
	nt.RootRepos[configsync.RootSyncName].Add(nomostest.StructuredNSPath(backendNamespace, configsync.RepoSyncName), repoSyncBackend)
	nt.RootRepos[configsync.RootSyncName].CommitAndPush("Update backend RepoSync git sync depth to 0")
	nt.WaitForRepoSyncs()

	// Verify the ns-reconciler-backend-git-sync ConfigMap has the correct git sync depth setting.
	_, err = nomostest.Retry(30*time.Second, func() error {
		return nt.Validate(nsReconcilerBackendGitSyncCM, v1.NSConfigManagementSystem, &corev1.ConfigMap{},
			nomostest.HasKeyValuePairInConfigMapData(key, "0"))
	})
	if err != nil {
		nt.T.Fatal(err)
	}

	// Clear `spec.override` from repoSyncBackend
	repoSyncBackend.Spec.Override = v1beta1.OverrideSpec{}
	nt.RootRepos[configsync.RootSyncName].Add(nomostest.StructuredNSPath(backendNamespace, configsync.RepoSyncName), repoSyncBackend)
	nt.RootRepos[configsync.RootSyncName].CommitAndPush("Clear `spec.override` from repoSyncBackend")
	nt.WaitForRepoSyncs()

	// Verify the ns-reconciler-backend-git-sync ConfigMap has the correct git sync depth setting.
	_, err = nomostest.Retry(30*time.Second, func() error {
		return nt.Validate(nsReconcilerBackendGitSyncCM, v1.NSConfigManagementSystem, &corev1.ConfigMap{},
			nomostest.HasKeyValuePairInConfigMapData(key, defaultDepth))
	})
	if err != nil {
		nt.T.Fatal(err)
	}

	// Verify the root-reconciler-git-sync ConfigMap has the correct git sync depth setting.
	err = nt.Validate(rootReconcilerGitSyncCM, v1.NSConfigManagementSystem, &corev1.ConfigMap{},
		nomostest.HasKeyValuePairInConfigMapData(key, defaultDepth))
	if err != nil {
		nt.T.Fatal(err)
	}

	// Verify the ns-reconciler-frontend-git-sync ConfigMap has the correct git sync depth setting.
	err = nt.Validate(nsReconcilerFrontendGitSyncCM, v1.NSConfigManagementSystem, &corev1.ConfigMap{},
		nomostest.HasKeyValuePairInConfigMapData(key, defaultDepth))
	if err != nil {
		nt.T.Fatal(err)
	}

	err = nt.ValidateMetrics(nomostest.SyncMetricsToLatestCommit(nt), func() error {
		return nt.ValidateMetricNotFound(ocmetrics.GitSyncDepthOverrideCountView.Name)
	})
	if err != nil {
		nt.T.Fatal(err)
	}
}
