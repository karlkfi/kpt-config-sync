package e2e

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/nomos/e2e/nomostest"
	"github.com/google/nomos/e2e/nomostest/metrics"
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

	err := nt.RetryMetrics(60*time.Second, func(prev metrics.ConfigSyncMetrics) error {
		nt.ParseMetrics(prev)
		// Validate parse error metric is emitted.
		err := nt.ValidateParseErrors(reconciler.RootSyncName, status.SourceErrorCode)
		if err != nil {
			return err
		}
		// Validate reconciler error metric is emitted.
		return nt.ValidateReconcilerErrors(reconciler.RootSyncName, "source")
	})
	if err != nil {
		t.Errorf("validating metrics: %v", err)
	}

	// Update RootSync to valid branch name
	rs = fake.RootSyncObject()
	nt.MustMergePatch(rs, `{"spec": {"git": {"branch": "main"}}}`)

	nt.WaitForRepoSyncs()

	// Validate no error metrics are emitted.
	// TODO(b/162601559): internal_errors_total metric from diff.go
	//err = nt.RetryMetrics(60*time.Second, func(prev metrics.ConfigSyncMetrics) error {
	//	nt.ParseMetrics(prev)
	//	return nt.ValidateErrorMetricsNotFound()
	//})
	//if err != nil {
	//	t.Errorf("validating error metrics: %v", err)
	//}
}

func TestInvalidRepoSyncBranchStatus(t *testing.T) {
	nt := nomostest.New(t, ntopts.SkipMonoRepo, ntopts.NamespaceRepo(namespaceRepo))

	rs := nomostest.RepoSyncObject(namespaceRepo)
	rs.Spec.Branch = "invalid-branch"
	nt.Root.Add(nomostest.StructuredNSPath(namespaceRepo, nomostest.RepoSyncFileName), rs)
	nt.Root.CommitAndPush("Update RepoSync to invalid branch name")

	nt.WaitForRepoSyncSourceError(namespaceRepo, status.SourceErrorCode)

	err := nt.RetryMetrics(60*time.Second, func(prev metrics.ConfigSyncMetrics) error {
		nt.ParseMetrics(prev)
		// Validate parse error metric is emitted.
		err := nt.ValidateParseErrors(reconciler.RootSyncName, status.SourceErrorCode)
		if err != nil {
			t.Errorf("validating parse_errors_total metric: %v", err)
		}
		// Validate reconciler error metric is emitted.
		return nt.ValidateReconcilerErrors(reconciler.RootSyncName, "source")
	})
	if err != nil {
		t.Errorf("validating metrics: %v", err)
	}

	rs.Spec.Branch = nomostest.MainBranch
	nt.Root.Add(nomostest.StructuredNSPath(namespaceRepo, nomostest.RepoSyncFileName), rs)
	nt.Root.CommitAndPush("Update RepoSync to valid branch name")

	nt.WaitForRepoSyncs()

	// Validate no error metrics are emitted.
	// TODO(b/162601559): internal_errors_total metric from diff.go
	//err = nt.RetryMetrics(60*time.Second, func(prev metrics.ConfigSyncMetrics) error {
	//	nt.ParseMetrics(prev)
	//	return nt.ValidateErrorMetricsNotFound()
	//})
	//if err != nil {
	//	t.Errorf("validating error metrics: %v", err)
	//}
}

func TestInvalidMonoRepoBranchStatus(t *testing.T) {
	nt := nomostest.New(t, ntopts.SkipMultiRepo)

	resetGitBranch(nt, "invalid-branch")

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
		nomostest.DeletePodByLabel(nt, "app", "reconciler-manager")
	} else {
		nomostest.DeletePodByLabel(nt, "app", "git-importer")
		nomostest.DeletePodByLabel(nt, "app", "monitor")
	}
}
