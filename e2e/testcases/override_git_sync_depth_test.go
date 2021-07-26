package e2e

import (
	"testing"
	"time"

	"github.com/google/nomos/e2e/nomostest"
	"github.com/google/nomos/e2e/nomostest/ntopts"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	"github.com/google/nomos/pkg/api/configsync/v1beta1"
	"github.com/google/nomos/pkg/reconcilermanager"
	"github.com/google/nomos/pkg/reconcilermanager/controllers"
	"github.com/google/nomos/pkg/testing/fake"
	corev1 "k8s.io/api/core/v1"
)

func TestOverrideGitSyncDepthV1Alpha1(t *testing.T) {
	nt := nomostest.New(t, ntopts.SkipMonoRepo, ntopts.NamespaceRepo(backendNamespace), ntopts.NamespaceRepo(frontendNamespace))
	nt.WaitForRepoSyncs()

	rootReconcilerGitSyncCM := controllers.RootSyncResourceName(reconcilermanager.GitSync)
	nsReconcilerBackendGitSyncCM := controllers.RepoSyncResourceName(backendNamespace, reconcilermanager.GitSync)
	nsReconcilerFrontendGitSyncCM := controllers.RepoSyncResourceName(frontendNamespace, reconcilermanager.GitSync)
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

	rootSync := fake.RootSyncObject()
	repoSyncBackend := nomostest.RepoSyncObject(backendNamespace)

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
	nt.Root.Add(nomostest.StructuredNSPath(backendNamespace, nomostest.RepoSyncFileName), repoSyncBackend)
	nt.Root.CommitAndPush("Update backend RepoSync git sync depth to 33")
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
	nt.Root.Add(nomostest.StructuredNSPath(backendNamespace, nomostest.RepoSyncFileName), repoSyncBackend)
	nt.Root.CommitAndPush("Update backend RepoSync git sync depth to 0")
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
	nt.Root.Add(nomostest.StructuredNSPath(backendNamespace, nomostest.RepoSyncFileName), repoSyncBackend)
	nt.Root.CommitAndPush("Clear `spec.override` from repoSyncBackend")
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
}

func TestOverrideGitSyncDepthV1Beta1(t *testing.T) {
	nt := nomostest.New(t, ntopts.SkipMonoRepo, ntopts.NamespaceRepo(backendNamespace), ntopts.NamespaceRepo(frontendNamespace))
	nt.WaitForRepoSyncs()

	rootReconcilerGitSyncCM := controllers.RootSyncResourceName(reconcilermanager.GitSync)
	nsReconcilerBackendGitSyncCM := controllers.RepoSyncResourceName(backendNamespace, reconcilermanager.GitSync)
	nsReconcilerFrontendGitSyncCM := controllers.RepoSyncResourceName(frontendNamespace, reconcilermanager.GitSync)
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

	rootSync := fake.RootSyncObjectV1Beta1()
	repoSyncBackend := nomostest.RepoSyncObjectV1Beta1(backendNamespace)

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
	nt.Root.Add(nomostest.StructuredNSPath(backendNamespace, nomostest.RepoSyncFileName), repoSyncBackend)
	nt.Root.CommitAndPush("Update backend RepoSync git sync depth to 33")
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
	nt.Root.Add(nomostest.StructuredNSPath(backendNamespace, nomostest.RepoSyncFileName), repoSyncBackend)
	nt.Root.CommitAndPush("Update backend RepoSync git sync depth to 0")
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
	nt.Root.Add(nomostest.StructuredNSPath(backendNamespace, nomostest.RepoSyncFileName), repoSyncBackend)
	nt.Root.CommitAndPush("Clear `spec.override` from repoSyncBackend")
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
}
