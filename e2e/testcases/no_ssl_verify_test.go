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
	ocmetrics "github.com/google/nomos/pkg/metrics"
	"github.com/google/nomos/pkg/reconciler"
	"github.com/google/nomos/pkg/reconcilermanager"
	"github.com/google/nomos/pkg/reconcilermanager/controllers"
	"github.com/google/nomos/pkg/testing/fake"
	corev1 "k8s.io/api/core/v1"
)

func TestNoSSLVerifyV1Alpha1(t *testing.T) {
	nt := nomostest.New(t, ntopts.SkipMonoRepo,
		ntopts.NamespaceRepo(backendNamespace, configsync.RepoSyncName),
		ntopts.NamespaceRepo(frontendNamespace, configsync.RepoSyncName))
	nt.WaitForRepoSyncs()

	rootReconcilerGitSyncCM := controllers.ReconcilerResourceName(nomostest.DefaultRootReconcilerName, reconcilermanager.GitSync)
	nsReconcilerBackendGitSyncCM := controllers.ReconcilerResourceName(reconciler.NsReconcilerName(backendNamespace, configsync.RepoSyncName), reconcilermanager.GitSync)
	nsReconcilerFrontendGitSyncCM := controllers.ReconcilerResourceName(reconciler.NsReconcilerName(frontendNamespace, configsync.RepoSyncName), reconcilermanager.GitSync)
	key := "GIT_SSL_NO_VERIFY"
	value := "true"

	// Verify the root-reconciler-git-sync ConfigMap does not have the key.
	err := nt.Validate(rootReconcilerGitSyncCM, v1.NSConfigManagementSystem, &corev1.ConfigMap{}, nomostest.MissingKeyInConfigMapData(key))
	if err != nil {
		nt.T.Fatal(err)
	}

	// Verify the ns-reconciler-backend-git-sync ConfigMap does not have the key.
	err = nt.Validate(nsReconcilerBackendGitSyncCM, v1.NSConfigManagementSystem, &corev1.ConfigMap{}, nomostest.MissingKeyInConfigMapData(key))
	if err != nil {
		nt.T.Fatal(err)
	}

	// Verify the ns-reconciler-frontend-git-sync ConfigMap does not have the key.
	err = nt.Validate(nsReconcilerFrontendGitSyncCM, v1.NSConfigManagementSystem, &corev1.ConfigMap{}, nomostest.MissingKeyInConfigMapData(key))
	if err != nil {
		nt.T.Fatal(err)
	}

	err = nt.ValidateMetrics(nomostest.SyncMetricsToLatestCommit(nt), func() error {
		return nt.ValidateMetricNotFound(ocmetrics.NoSSLVerifyCountView.Name)
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

	// Set noSSLVerify to true for root-reconciler
	nt.MustMergePatch(rootSync, `{"spec": {"git": {"noSSLVerify": true}}}`)

	// Verify the root-reconciler-git-sync ConfigMap has the correct git sync depth setting.
	_, err = nomostest.Retry(30*time.Second, func() error {
		return nt.Validate(rootReconcilerGitSyncCM, v1.NSConfigManagementSystem, &corev1.ConfigMap{},
			nomostest.HasKeyValuePairInConfigMapData(key, value))
	})
	if err != nil {
		nt.T.Fatal(err)
	}

	// Verify the ns-reconciler-backend-git-sync ConfigMap has the correct git sync depth setting.
	err = nt.Validate(nsReconcilerBackendGitSyncCM, v1.NSConfigManagementSystem, &corev1.ConfigMap{}, nomostest.MissingKeyInConfigMapData(key))
	if err != nil {
		nt.T.Fatal(err)
	}

	// Verify the ns-reconciler-frontend-git-sync ConfigMap has the correct git sync depth setting.
	err = nt.Validate(nsReconcilerFrontendGitSyncCM, v1.NSConfigManagementSystem, &corev1.ConfigMap{}, nomostest.MissingKeyInConfigMapData(key))
	if err != nil {
		nt.T.Fatal(err)
	}

	// Set noSSLVerify to true for ns-reconciler-backend
	repoSyncBackend.Spec.NoSSLVerify = true
	nt.RootRepos[configsync.RootSyncName].Add(nomostest.StructuredNSPath(backendNamespace, configsync.RepoSyncName), repoSyncBackend)
	nt.RootRepos[configsync.RootSyncName].CommitAndPush("Update backend RepoSync NoSSLVerify to true")
	nt.WaitForRepoSyncs()

	// Verify the ns-reconciler-backend-git-sync ConfigMap has the correct git sync depth setting.
	_, err = nomostest.Retry(30*time.Second, func() error {
		return nt.Validate(nsReconcilerBackendGitSyncCM, v1.NSConfigManagementSystem, &corev1.ConfigMap{},
			nomostest.HasKeyValuePairInConfigMapData(key, value))
	})
	if err != nil {
		nt.T.Fatal(err)
	}

	// Verify the root-reconciler-git-sync ConfigMap has the correct git sync depth setting.
	err = nt.Validate(rootReconcilerGitSyncCM, v1.NSConfigManagementSystem, &corev1.ConfigMap{},
		nomostest.HasKeyValuePairInConfigMapData(key, value))
	if err != nil {
		nt.T.Fatal(err)
	}

	err = nt.ValidateMetrics(nomostest.SyncMetricsToLatestCommit(nt), func() error {
		return nt.ValidateNoSSLVerifyCount(2)
	})
	if err != nil {
		nt.T.Fatal(err)
	}

	// Set noSSLVerify to false for root-reconciler
	nt.MustMergePatch(rootSync, `{"spec": {"git": {"noSSLVerify": false}}}`)

	// Verify the root-reconciler-git-sync ConfigMap has the correct git sync depth setting.
	_, err = nomostest.Retry(30*time.Second, func() error {
		return nt.Validate(rootReconcilerGitSyncCM, v1.NSConfigManagementSystem, &corev1.ConfigMap{}, nomostest.MissingKeyInConfigMapData(key))
	})
	if err != nil {
		nt.T.Fatal(err)
	}

	// Set noSSLVerify to false from repoSyncBackend
	repoSyncBackend.Spec.NoSSLVerify = false
	nt.RootRepos[configsync.RootSyncName].Add(nomostest.StructuredNSPath(backendNamespace, configsync.RepoSyncName), repoSyncBackend)
	nt.RootRepos[configsync.RootSyncName].CommitAndPush("Update backend RepoSync NoSSLVerify to false")
	nt.WaitForRepoSyncs()

	// Verify the ns-reconciler-backend-git-sync ConfigMap has the correct git sync depth setting.
	_, err = nomostest.Retry(30*time.Second, func() error {
		return nt.Validate(nsReconcilerBackendGitSyncCM, v1.NSConfigManagementSystem, &corev1.ConfigMap{}, nomostest.MissingKeyInConfigMapData(key))
	})
	if err != nil {
		nt.T.Fatal(err)
	}

	// Verify the root-reconciler-git-sync ConfigMap has the correct git sync depth setting.
	err = nt.Validate(rootReconcilerGitSyncCM, v1.NSConfigManagementSystem, &corev1.ConfigMap{}, nomostest.MissingKeyInConfigMapData(key))
	if err != nil {
		nt.T.Fatal(err)
	}

	// Verify the ns-reconciler-frontend-git-sync ConfigMap has the correct git sync depth setting.
	err = nt.Validate(nsReconcilerFrontendGitSyncCM, v1.NSConfigManagementSystem, &corev1.ConfigMap{}, nomostest.MissingKeyInConfigMapData(key))
	if err != nil {
		nt.T.Fatal(err)
	}

	err = nt.ValidateMetrics(nomostest.SyncMetricsToLatestCommit(nt), func() error {
		return nt.ValidateMetricNotFound(ocmetrics.NoSSLVerifyCountView.Name)
	})
	if err != nil {
		nt.T.Fatal(err)
	}
}

func TestNoSSLVerifyV1Beta1(t *testing.T) {
	nt := nomostest.New(t, ntopts.SkipMonoRepo,
		ntopts.NamespaceRepo(backendNamespace, configsync.RepoSyncName),
		ntopts.NamespaceRepo(frontendNamespace, configsync.RepoSyncName))
	nt.WaitForRepoSyncs()

	rootReconcilerGitSyncCM := controllers.ReconcilerResourceName(nomostest.DefaultRootReconcilerName, reconcilermanager.GitSync)
	nsReconcilerBackendGitSyncCM := controllers.ReconcilerResourceName(reconciler.NsReconcilerName(backendNamespace, configsync.RepoSyncName), reconcilermanager.GitSync)
	nsReconcilerFrontendGitSyncCM := controllers.ReconcilerResourceName(reconciler.NsReconcilerName(frontendNamespace, configsync.RepoSyncName), reconcilermanager.GitSync)
	key := "GIT_SSL_NO_VERIFY"
	value := "true"

	// Verify the root-reconciler-git-sync ConfigMap does not have the key.
	err := nt.Validate(rootReconcilerGitSyncCM, v1.NSConfigManagementSystem, &corev1.ConfigMap{}, nomostest.MissingKeyInConfigMapData(key))
	if err != nil {
		nt.T.Fatal(err)
	}

	// Verify the ns-reconciler-backend-git-sync ConfigMap does not have the key.
	err = nt.Validate(nsReconcilerBackendGitSyncCM, v1.NSConfigManagementSystem, &corev1.ConfigMap{}, nomostest.MissingKeyInConfigMapData(key))
	if err != nil {
		nt.T.Fatal(err)
	}

	// Verify the ns-reconciler-frontend-git-sync ConfigMap does not have the key.
	err = nt.Validate(nsReconcilerFrontendGitSyncCM, v1.NSConfigManagementSystem, &corev1.ConfigMap{}, nomostest.MissingKeyInConfigMapData(key))
	if err != nil {
		nt.T.Fatal(err)
	}

	err = nt.ValidateMetrics(nomostest.SyncMetricsToLatestCommit(nt), func() error {
		return nt.ValidateMetricNotFound(ocmetrics.NoSSLVerifyCountView.Name)
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

	// Set noSSLVerify to true for root-reconciler
	nt.MustMergePatch(rootSync, `{"spec": {"git": {"noSSLVerify": true}}}`)

	// Verify the root-reconciler-git-sync ConfigMap has the correct git sync depth setting.
	_, err = nomostest.Retry(30*time.Second, func() error {
		return nt.Validate(rootReconcilerGitSyncCM, v1.NSConfigManagementSystem, &corev1.ConfigMap{},
			nomostest.HasKeyValuePairInConfigMapData(key, value))
	})
	if err != nil {
		nt.T.Fatal(err)
	}

	// Verify the ns-reconciler-backend-git-sync ConfigMap has the correct git sync depth setting.
	err = nt.Validate(nsReconcilerBackendGitSyncCM, v1.NSConfigManagementSystem, &corev1.ConfigMap{}, nomostest.MissingKeyInConfigMapData(key))
	if err != nil {
		nt.T.Fatal(err)
	}

	// Verify the ns-reconciler-frontend-git-sync ConfigMap has the correct git sync depth setting.
	err = nt.Validate(nsReconcilerFrontendGitSyncCM, v1.NSConfigManagementSystem, &corev1.ConfigMap{}, nomostest.MissingKeyInConfigMapData(key))
	if err != nil {
		nt.T.Fatal(err)
	}

	// Set noSSLVerify to true for ns-reconciler-backend
	repoSyncBackend.Spec.NoSSLVerify = true
	nt.RootRepos[configsync.RootSyncName].Add(nomostest.StructuredNSPath(backendNamespace, configsync.RepoSyncName), repoSyncBackend)
	nt.RootRepos[configsync.RootSyncName].CommitAndPush("Update backend RepoSync NoSSLVerify to true")
	nt.WaitForRepoSyncs()

	// Verify the ns-reconciler-backend-git-sync ConfigMap has the correct git sync depth setting.
	_, err = nomostest.Retry(30*time.Second, func() error {
		return nt.Validate(nsReconcilerBackendGitSyncCM, v1.NSConfigManagementSystem, &corev1.ConfigMap{},
			nomostest.HasKeyValuePairInConfigMapData(key, value))
	})
	if err != nil {
		nt.T.Fatal(err)
	}

	// Verify the root-reconciler-git-sync ConfigMap has the correct git sync depth setting.
	err = nt.Validate(rootReconcilerGitSyncCM, v1.NSConfigManagementSystem, &corev1.ConfigMap{},
		nomostest.HasKeyValuePairInConfigMapData(key, value))
	if err != nil {
		nt.T.Fatal(err)
	}

	err = nt.ValidateMetrics(nomostest.SyncMetricsToLatestCommit(nt), func() error {
		return nt.ValidateNoSSLVerifyCount(2)
	})
	if err != nil {
		nt.T.Fatal(err)
	}

	// Set noSSLVerify to false for root-reconciler
	nt.MustMergePatch(rootSync, `{"spec": {"git": {"noSSLVerify": false}}}`)

	// Verify the root-reconciler-git-sync ConfigMap has the correct git sync depth setting.
	_, err = nomostest.Retry(30*time.Second, func() error {
		return nt.Validate(rootReconcilerGitSyncCM, v1.NSConfigManagementSystem, &corev1.ConfigMap{}, nomostest.MissingKeyInConfigMapData(key))
	})
	if err != nil {
		nt.T.Fatal(err)
	}

	// Set noSSLVerify to false from repoSyncBackend
	repoSyncBackend.Spec.NoSSLVerify = false
	nt.RootRepos[configsync.RootSyncName].Add(nomostest.StructuredNSPath(backendNamespace, configsync.RepoSyncName), repoSyncBackend)
	nt.RootRepos[configsync.RootSyncName].CommitAndPush("Update backend RepoSync NoSSLVerify to false")
	nt.WaitForRepoSyncs()

	// Verify the ns-reconciler-backend-git-sync ConfigMap has the correct git sync depth setting.
	_, err = nomostest.Retry(30*time.Second, func() error {
		return nt.Validate(nsReconcilerBackendGitSyncCM, v1.NSConfigManagementSystem, &corev1.ConfigMap{}, nomostest.MissingKeyInConfigMapData(key))
	})
	if err != nil {
		nt.T.Fatal(err)
	}

	// Verify the root-reconciler-git-sync ConfigMap has the correct git sync depth setting.
	err = nt.Validate(rootReconcilerGitSyncCM, v1.NSConfigManagementSystem, &corev1.ConfigMap{}, nomostest.MissingKeyInConfigMapData(key))
	if err != nil {
		nt.T.Fatal(err)
	}

	// Verify the ns-reconciler-frontend-git-sync ConfigMap has the correct git sync depth setting.
	err = nt.Validate(nsReconcilerFrontendGitSyncCM, v1.NSConfigManagementSystem, &corev1.ConfigMap{}, nomostest.MissingKeyInConfigMapData(key))
	if err != nil {
		nt.T.Fatal(err)
	}

	err = nt.ValidateMetrics(nomostest.SyncMetricsToLatestCommit(nt), func() error {
		return nt.ValidateMetricNotFound(ocmetrics.NoSSLVerifyCountView.Name)
	})
	if err != nil {
		nt.T.Fatal(err)
	}
}
