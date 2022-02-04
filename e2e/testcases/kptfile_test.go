package e2e

import (
	"testing"

	"github.com/google/nomos/e2e/nomostest"
	"github.com/google/nomos/e2e/nomostest/metrics"
	"github.com/google/nomos/pkg/api/configsync"
	"github.com/google/nomos/pkg/testing/fake"
)

func TestIgnoreKptfiles(t *testing.T) {
	nt := nomostest.New(t)

	// Add multiple Kptfiles
	nt.RootRepos[configsync.RootSyncName].AddFile("acme/cluster/Kptfile", []byte("random content"))
	nt.RootRepos[configsync.RootSyncName].AddFile("acme/namespaces/foo/Kptfile", nil)
	nt.RootRepos[configsync.RootSyncName].AddFile("acme/namespaces/foo/subdir/Kptfile", []byte("# some comment"))
	nt.RootRepos[configsync.RootSyncName].Add("acme/namespaces/foo/ns.yaml", fake.NamespaceObject("foo"))
	nt.RootRepos[configsync.RootSyncName].CommitAndPush("Adding multiple Kptfiles")
	nt.WaitForRepoSyncs()
	nt.RenewClient()

	err := nt.Validate("foo", "", fake.NamespaceObject("foo"))
	if err != nil {
		nt.T.Fatal(err)
	}

	// Validate multi-repo metrics.
	err = nt.ValidateMetrics(nomostest.SyncMetricsToLatestCommit(nt), func() error {
		err = nt.ValidateMultiRepoMetrics(nomostest.DefaultRootReconcilerName, 2,
			metrics.ResourceCreated("Namespace"))
		if err != nil {
			return err
		}
		// Validate no error metrics are emitted.
		// TODO(b/162601559): internal_errors_total metric from diff.go
		//return nt.ValidateErrorMetricsNotFound()
		return nil
	})
	if err != nil {
		nt.T.Errorf("validating metrics: %v", err)
	}
}
