package e2e

import (
	"testing"
	"time"

	"github.com/google/nomos/e2e/nomostest"
	"github.com/google/nomos/e2e/nomostest/metrics"
	"github.com/google/nomos/pkg/reconciler"
	"github.com/google/nomos/pkg/testing/fake"
)

func TestIgnoreKptfiles(t *testing.T) {
	nt := nomostest.New(t)

	// Add multiple Kptfiles
	nt.Root.AddFile("acme/cluster/Kptfile", []byte("random content"))
	nt.Root.AddFile("acme/namespaces/foo/Kptfile", nil)
	nt.Root.AddFile("acme/namespaces/foo/subdir/Kptfile", []byte("# some comment"))
	nt.Root.Add("acme/namespaces/foo/ns.yaml", fake.NamespaceObject("foo"))
	nt.Root.CommitAndPush("Adding multiple Kptfiles")
	nt.WaitForRepoSyncs()
	nt.RenewClient()

	err := nt.Validate("foo", "", fake.NamespaceObject("foo"))
	if err != nil {
		nt.T.Fatal(err)
	}

	// Validate multi-repo metrics.
	err = nt.RetryMetrics(60*time.Second, func(prev metrics.ConfigSyncMetrics) error {
		nt.ParseMetrics(prev)
		err = nt.ValidateMultiRepoMetrics(reconciler.RootSyncName, 2,
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
