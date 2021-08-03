package e2e

import (
	"testing"
	"time"

	"github.com/google/nomos/e2e/nomostest"
	"github.com/google/nomos/e2e/nomostest/ntopts"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/reconcilermanager"
	"github.com/google/nomos/pkg/reconcilermanager/controllers"
	"github.com/google/nomos/pkg/testing/fake"
	corev1 "k8s.io/api/core/v1"
)

func TestNoSSLVerifyV1Alpha1(t *testing.T) {
	nt := nomostest.New(t, ntopts.SkipMonoRepo, ntopts.NamespaceRepo(backendNamespace), ntopts.NamespaceRepo(frontendNamespace))
	nt.WaitForRepoSyncs()

	rootReconcilerGitSyncCM := controllers.RootSyncResourceName(reconcilermanager.GitSync)
	nsReconcilerBackendGitSyncCM := controllers.RepoSyncResourceName(backendNamespace, reconcilermanager.GitSync)
	nsReconcilerFrontendGitSyncCM := controllers.RepoSyncResourceName(frontendNamespace, reconcilermanager.GitSync)
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

	repo, exist := nt.NonRootRepos[backendNamespace]
	if !exist {
		nt.T.Fatal("nonexistent repo")
	}
	rootSync := fake.RootSyncObject()
	repoSyncBackend := nomostest.RepoSyncObject(backendNamespace, nt.GitProvider.SyncURL(repo.RemoteRepoName))

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
	nt.Root.Add(nomostest.StructuredNSPath(backendNamespace, nomostest.RepoSyncFileName), repoSyncBackend)
	nt.Root.CommitAndPush("Update backend RepoSync NoSSLVerify to true")
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
	nt.Root.Add(nomostest.StructuredNSPath(backendNamespace, nomostest.RepoSyncFileName), repoSyncBackend)
	nt.Root.CommitAndPush("Update backend RepoSync NoSSLVerify to false")
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
}

func TestNoSSLVerifyV1Beta1(t *testing.T) {
	nt := nomostest.New(t, ntopts.SkipMonoRepo, ntopts.NamespaceRepo(backendNamespace), ntopts.NamespaceRepo(frontendNamespace))
	nt.WaitForRepoSyncs()

	rootReconcilerGitSyncCM := controllers.RootSyncResourceName(reconcilermanager.GitSync)
	nsReconcilerBackendGitSyncCM := controllers.RepoSyncResourceName(backendNamespace, reconcilermanager.GitSync)
	nsReconcilerFrontendGitSyncCM := controllers.RepoSyncResourceName(frontendNamespace, reconcilermanager.GitSync)
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

	repo, exist := nt.NonRootRepos[backendNamespace]
	if !exist {
		nt.T.Fatal("nonexistent repo")
	}

	rootSync := fake.RootSyncObjectV1Beta1()
	repoSyncBackend := nomostest.RepoSyncObjectV1Beta1(backendNamespace, nt.GitProvider.SyncURL(repo.RemoteRepoName))

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
	nt.Root.Add(nomostest.StructuredNSPath(backendNamespace, nomostest.RepoSyncFileName), repoSyncBackend)
	nt.Root.CommitAndPush("Update backend RepoSync NoSSLVerify to true")
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
	nt.Root.Add(nomostest.StructuredNSPath(backendNamespace, nomostest.RepoSyncFileName), repoSyncBackend)
	nt.Root.CommitAndPush("Update backend RepoSync NoSSLVerify to false")
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
}
