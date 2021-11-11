package e2e

import (
	"fmt"
	"testing"

	"github.com/google/nomos/e2e/nomostest"
	"github.com/google/nomos/e2e/nomostest/ntopts"
	"github.com/google/nomos/pkg/api/configmanagement"
	"github.com/google/nomos/pkg/reconciler"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/testing/fake"
	corev1 "k8s.io/api/core/v1"
)

func TestInvalidRootSyncBranchStatus(t *testing.T) {
	nt := nomostest.New(t, ntopts.SkipMonoRepo)

	// Update RootSync to invalid branch name
	rs := fake.RootSyncObject()
	nt.MustMergePatch(rs, `{"spec": {"git": {"branch": "invalid-branch"}}}`)

	nt.WaitForRootSyncSourceError(status.SourceErrorCode)

	err := nt.ValidateMetrics(nomostest.SyncMetricsToReconcilerSourceError(reconciler.RootSyncName), func() error {
		// Validate reconciler error metric is emitted.
		return nt.ValidateReconcilerErrors(reconciler.RootSyncName, "source")
	})
	if err != nil {
		nt.T.Errorf("validating metrics: %v", err)
	}

	// Update RootSync to valid branch name
	rs = fake.RootSyncObject()
	nt.MustMergePatch(rs, `{"spec": {"git": {"branch": "main"}}}`)

	nt.WaitForRepoSyncs()

	// Validate no error metrics are emitted.
	// TODO(b/162601559): internal_errors_total metric from diff.go
	//err = nt.ValidateMetrics(nomostest.MetricsLatestCommit, func() error {
	//	return nt.ValidateErrorMetricsNotFound()
	//})
	//if err != nil {
	//	nt.T.Errorf("validating error metrics: %v", err)
	//}
}

func TestInvalidRepoSyncBranchStatus(t *testing.T) {
	nt := nomostest.New(t, ntopts.SkipMonoRepo, ntopts.NamespaceRepo(namespaceRepo))

	repo, exist := nt.NonRootRepos[namespaceRepo]
	if !exist {
		nt.T.Fatal("nonexistent repo")
	}
	rs := nomostest.RepoSyncObject(namespaceRepo, nt.GitProvider.SyncURL(repo.RemoteRepoName))
	rs.Spec.Branch = "invalid-branch"
	nt.Root.Add(nomostest.StructuredNSPath(namespaceRepo, nomostest.RepoSyncFileName), rs)
	nt.Root.CommitAndPush("Update RepoSync to invalid branch name")

	nt.WaitForRepoSyncSourceError(namespaceRepo, status.SourceErrorCode)

	err := nt.ValidateMetrics(nomostest.SyncMetricsToLatestCommit(nt), func() error {
		// Validate reconciler error metric is emitted.
		return nt.ValidateReconcilerErrors(reconciler.RootSyncName, "source")
	})
	if err != nil {
		nt.T.Errorf("validating metrics: %v", err)
	}

	rs.Spec.Branch = nomostest.MainBranch
	nt.Root.Add(nomostest.StructuredNSPath(namespaceRepo, nomostest.RepoSyncFileName), rs)
	nt.Root.CommitAndPush("Update RepoSync to valid branch name")

	nt.WaitForRepoSyncs()

	// Validate no error metrics are emitted.
	// TODO(b/162601559): internal_errors_total metric from diff.go
	//err = nt.ValidateMetrics(nomostest.MetricsLatestCommit, func() error {
	//	return nt.ValidateErrorMetricsNotFound()
	//})
	//if err != nil {
	//	nt.T.Errorf("validating error metrics: %v", err)
	//}
}

func TestInvalidMonoRepoBranchStatus(t *testing.T) {
	nt := nomostest.New(t, ntopts.SkipMultiRepo)

	resetGitBranch(nt, "invalid-branch")

	nt.WaitForRepoSourceError(status.SourceErrorCode)

	resetGitBranch(nt, "main")
	nt.WaitForRepoSourceErrorClear()
}

func TestSyncFailureAfterSuccessfulSyncs(t *testing.T) {
	nt := nomostest.New(t)

	// Add audit namespace.
	auditNS := "audit"
	// The test will delete the branch later, but the main branch can't be deleted
	// on some Git providers (e.g. Bitbucket), so using a develop branch.
	devBranch := "develop"
	nt.Root.CreateBranch(devBranch)
	nt.Root.CheckoutBranch(devBranch)
	nt.Root.Add(fmt.Sprintf("acme/namespaces/%s/ns.yaml", auditNS),
		fake.NamespaceObject(auditNS))
	nt.Root.CommitAndPushBranch("add namespace to acme directory", devBranch)

	// Update RootSync to sync from the dev branch
	if nt.MultiRepo {
		rs := fake.RootSyncObject()
		nt.MustMergePatch(rs, fmt.Sprintf(`{"spec": {"git": {"branch": "%s"}}}`, devBranch))
	} else {
		resetGitBranch(nt, devBranch)
	}
	nt.WaitForRepoSyncs()

	// Validate namespace 'acme' created.
	err := nt.Validate(auditNS, "", fake.NamespaceObject(auditNS))
	if err != nil {
		nt.T.Error(err)
	}

	// Make the sync fail by invalidating the source repo.
	nt.Root.RenameBranch(devBranch, "invalid-branch")
	if nt.MultiRepo {
		nt.WaitForRootSyncSourceError(status.SourceErrorCode)
	} else {
		nt.WaitForRepoSourceError(status.SourceErrorCode)
	}

	// Change the remote branch name back to the original name.
	nt.Root.RenameBranch("invalid-branch", devBranch)
	nt.WaitForRepoSyncs()
	// Reset RootSync because the cleanup stage will check if RootSync is synced from the main branch.
	if nt.MultiRepo {
		rs := fake.RootSyncObject()
		nt.MustMergePatch(rs, fmt.Sprintf(`{"spec": {"git": {"branch": "%s"}}}`, nomostest.MainBranch))
	} else {
		resetGitBranch(nt, nomostest.MainBranch)
	}
}

// resetGitBranch updates GIT_SYNC_BRANCH in the config map and restart the reconcilers.
func resetGitBranch(nt *nomostest.NT, branch string) {
	nt.T.Logf("Change the GIT_SYNC_BRANCH name to %q", branch)
	cm := &corev1.ConfigMap{}
	err := nt.Get("git-sync", configmanagement.ControllerNamespace, cm)
	if err != nil {
		nt.T.Fatal(err)
	}
	nt.MustMergePatch(cm, fmt.Sprintf(`{"data":{"GIT_SYNC_BRANCH":"%s"}}`, branch))

	if nt.MultiRepo {
		nomostest.DeletePodByLabel(nt, "app", "reconciler-manager")
	} else {
		nomostest.DeletePodByLabel(nt, "app", "git-importer")
		nomostest.DeletePodByLabel(nt, "app", "monitor")
	}
}
