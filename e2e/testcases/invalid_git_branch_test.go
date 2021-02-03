package e2e

import (
	"fmt"
	"testing"

	"github.com/google/nomos/e2e/nomostest"
	"github.com/google/nomos/e2e/nomostest/ntopts"
	"github.com/google/nomos/pkg/api/configmanagement"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/testing/fake"
	corev1 "k8s.io/api/core/v1"
)

func TestInvalidRootSyncBranchStatus(t *testing.T) {
	nt := nomostest.New(t, ntopts.SkipMonoRepo)

	// Update RootSync to invalid branch name
	rs := fake.RootSyncObject()
	nt.MustMergePatch(rs, `{"spec": {"git": {"branch": "invalid-branch"}}}`)

	// Check for error code 2004 (this is a generic error code for the current impl, this may change if we
	// make better git error reporting.
	nt.WaitForRootSyncSourceError(status.SourceErrorCode)

	// Update RootSync to valid branch name
	rs = fake.RootSyncObject()
	nt.MustMergePatch(rs, `{"spec": {"git": {"branch": "main"}}}`)

	nt.WaitForRootSyncSourceErrorClear()
}

func TestInvalidRepoSyncBranchStatus(t *testing.T) {
	nt := nomostest.New(t, ntopts.SkipMonoRepo, ntopts.NamespaceRepo(namespaceRepo))

	rs := nomostest.RepoSyncObject(namespaceRepo)
	rs.Spec.Branch = "invalid-branch"
	nt.Root.Add(nomostest.StructuredNSPath(namespaceRepo, nomostest.RepoSyncFileName), rs)
	nt.Root.CommitAndPush("Update RepoSync to invalid branch name")

	// Check for error code 2004 (this is a generic error code for the current impl, this may change if we
	// make better git error reporting.
	nt.WaitForRepoSyncSourceError(namespaceRepo, status.SourceErrorCode)

	rs.Spec.Branch = nomostest.MainBranch
	nt.Root.Add(nomostest.StructuredNSPath(namespaceRepo, nomostest.RepoSyncFileName), rs)
	nt.Root.CommitAndPush("Update RepoSync to valid branch name")

	nt.WaitForRepoSyncSourceErrorClear(namespaceRepo)
}

func TestInvalidMonoRepoBranchStatus(t *testing.T) {
	nt := nomostest.New(t, ntopts.SkipMultiRepo)

	resetGitBranch(nt, "invalid-branch")

	// Check for error code 2004 (this is a generic error code for the current impl, this may change if we
	// make better git error reporting.
	nt.WaitForRepoSourceError(status.SourceErrorCode)

	resetGitBranch(nt, "main")
	nt.WaitForRepoSourceErrorClear()
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
		deletePodByLabel(nt, "app", "reconciler-manager")
	} else {
		deletePodByLabel(nt, "app", "git-importer")
		deletePodByLabel(nt, "app", "monitor")
	}
}
