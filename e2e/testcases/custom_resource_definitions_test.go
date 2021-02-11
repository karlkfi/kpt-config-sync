package e2e

import (
	"testing"
	"time"

	"github.com/google/nomos/e2e/nomostest"
	"github.com/google/nomos/e2e/nomostest/metrics"
	"github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical"
	"github.com/google/nomos/pkg/reconciler"
	"github.com/google/nomos/pkg/testing/fake"
)

func TestMustRemoveCustomResourceWithDefinition(t *testing.T) {
	nt := nomostest.New(t)

	nt.Root.Add("acme/cluster/anvil-crd.yaml", anvilV1CRD())
	nt.Root.Add("acme/namespaces/foo/ns.yaml", fake.NamespaceObject("foo"))
	nt.Root.Add("acme/namespaces/foo/anvil-v1.yaml", anvilCR("v1", "heavy", 10))
	nt.Root.CommitAndPush("Adding Anvil CRD and one Anvil CR")
	nt.WaitForRepoSyncs()
	nt.RenewClient()

	err := nt.Validate("heavy", "foo", anvilCR("v1", "", 0))
	if err != nil {
		t.Fatal(err)
	}

	// Validate multi-repo metrics.
	err = nt.RetryMetrics(60*time.Second, func(prev metrics.ConfigSyncMetrics) error {
		nt.ParseMetrics(prev)
		err = nt.ValidateMultiRepoMetrics(reconciler.RootSyncName, 3,
			metrics.ResourceCreated("Namespace"), metrics.ResourceCreated("CustomResourceDefinition"), metrics.ResourceCreated("Anvil"))
		if err != nil {
			return err
		}
		// Validate no error metrics are emitted.
		return nt.ValidateErrorMetricsNotFound()
	})
	if err != nil {
		t.Errorf("validating metrics: %v", err)
	}

	// This should cause an error.
	nt.Root.Remove("acme/cluster/anvil-crd.yaml")
	nt.Root.CommitAndPush("Removing Anvil CRD but leaving Anvil CR")

	if nt.MultiRepo {
		nt.WaitForRootSyncSourceError(nonhierarchical.UnsupportedCRDRemovalErrorCode)
	} else {
		nt.WaitForRepoImportErrorCode(nonhierarchical.UnsupportedCRDRemovalErrorCode)
	}

	err = nt.RetryMetrics(60*time.Second, func(prev metrics.ConfigSyncMetrics) error {
		nt.ParseMetrics(prev)
		// Validate parse error metric is emitted.
		err = nt.ValidateParseErrors(reconciler.RootSyncName, nonhierarchical.UnsupportedCRDRemovalErrorCode)
		if err != nil {
			return err
		}
		// Validate reconciler error metric is emitted.
		return nt.ValidateReconcilerErrors(reconciler.RootSyncName, "source")
	})
	if err != nil {
		t.Errorf("validating metrics: %v", err)
	}

	// This should fix the error.
	nt.Root.Remove("acme/namespaces/foo/anvil-v1.yaml")
	nt.Root.CommitAndPush("Removing the Anvil CR as well")
	nt.WaitForRepoSyncs()

	// Validate reconciler error is cleared.
	err = nt.RetryMetrics(60*time.Second, func(prev metrics.ConfigSyncMetrics) error {
		nt.ParseMetrics(prev)
		return nt.ValidateReconcilerErrors(reconciler.RootSyncName, "")
	})
	if err != nil {
		t.Errorf("validating reconciler_errors metric: %v", err)
	}
}
